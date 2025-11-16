#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[%s] %s\n' "$(date --iso-8601=seconds)" "$*"
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: $0 <bucket-name> [region]" >&2
  echo "Ex:    $0 talos-fgz fr-par" >&2
  exit 1
fi

BUCKET_NAME="$1"
REGION="${2:-fr-par}"

# Zones cibles (tu peux adapter)
ZONES=("fr-par-1" "fr-par-2" "fr-par-3")

MANIFEST_PATH="$(dirname "$0")/manifest.json"

if [[ ! -f "${MANIFEST_PATH}" ]]; then
  echo "ERREUR: manifest.json introuvable à ${MANIFEST_PATH}" >&2
  exit 1
fi

log "Lecture du manifest Packer..."
ARTIFACT_ID="$(jq -r '.builds[0].artifact_id' "${MANIFEST_PATH}")"

SRC_ZONE="${ARTIFACT_ID%%:*}"
SRC_IMAGE_ID="${ARTIFACT_ID##*:}"

log "Image Packer source:"
log "  zone      = ${SRC_ZONE}"
log "  image_id  = ${SRC_IMAGE_ID}"

log "Récupération des infos de l'image source..."
IMAGE_JSON="$(scw instance image get "${SRC_IMAGE_ID}" zone="${SRC_ZONE}" -o json)"
IMAGE_NAME="$(echo "${IMAGE_JSON}" | jq -r '.name')"

log "Nom de l'image source: ${IMAGE_NAME}"

# IMAGE_NAME ressemble à: talos-scaleway-v1.11.5-20251115195409
# On récupère juste la version vX.Y.Z
TALOS_VERSION="$(echo "${IMAGE_NAME}" | sed -E 's/^talos-scaleway-(v[0-9]+\.[0-9]+\.[0-9]+).*/\1/')"
if [[ -z "${TALOS_VERSION}" || "${TALOS_VERSION}" == "${IMAGE_NAME}" ]]; then
  echo "ERREUR: impossible d'extraire la version depuis le nom d'image '${IMAGE_NAME}'" >&2
  exit 1
fi

STABLE_IMAGE_NAME="talos-scaleway-${TALOS_VERSION}"
OBJECT_KEY="${IMAGE_NAME}.qcow2"

log "Version Talos détectée : ${TALOS_VERSION}"
log "Nom stable d'image     : ${STABLE_IMAGE_NAME}"

# Récupérer le snapshot racine (SBS) associé à l'image source
ROOT_SNAPSHOT_ID="$(echo "${IMAGE_JSON}" | jq -r '.root_volume.id // .image.root_volume.id')"
if [[ -z "${ROOT_SNAPSHOT_ID}" || "${ROOT_SNAPSHOT_ID}" == "null" ]]; then
  echo "ERREUR: impossible de récupérer root_volume.id pour l'image ${SRC_IMAGE_ID}" >&2
  exit 1
fi
log "Snapshot racine ID: ${ROOT_SNAPSHOT_ID}"

SNAPSHOT_SIZE="$(scw block snapshot get "${ROOT_SNAPSHOT_ID}" zone="${SRC_ZONE}" -o json | jq -r '.size')"
if [[ -z "${SNAPSHOT_SIZE}" || "${SNAPSHOT_SIZE}" == "null" ]]; then
  echo "ERREUR: impossible de récupérer la taille du snapshot ${ROOT_SNAPSHOT_ID}" >&2
  exit 1
fi
log "Taille du snapshot: ${SNAPSHOT_SIZE} bytes"

log "Export du snapshot ${ROOT_SNAPSHOT_ID} vers s3://${BUCKET_NAME}/${OBJECT_KEY}..."
scw block snapshot export-to-object-storage \
  snapshot-id="${ROOT_SNAPSHOT_ID}" \
  bucket="${BUCKET_NAME}" \
  key="${OBJECT_KEY}" \
  zone="${SRC_ZONE}"

log "Attente de la fin de l'export..."
scw block snapshot wait "${ROOT_SNAPSHOT_ID}" zone="${SRC_ZONE}" terminal-status=available
log "Export terminé."

# Pour chaque zone : on garantit une image 'talos-scaleway-vX.Y.Z'
for ZONE in "${ZONES[@]}"; do
  log "=== Traitement de la zone ${ZONE} ==="

  if [[ "${ZONE}" == "${SRC_ZONE}" ]]; then
    # Dans la zone source, on a déjà le snapshot; on l'utilise directement
    SNAP_ID="${ROOT_SNAPSHOT_ID}"
    log "Réutilisation du snapshot source ${SNAP_ID} dans ${ZONE}"
  else
    log "Import du snapshot dans ${ZONE} depuis s3://${BUCKET_NAME}/${OBJECT_KEY}..."
    IMPORT_JSON="$(
      scw block snapshot import-from-object-storage \
        bucket="${BUCKET_NAME}" \
        key="${OBJECT_KEY}" \
        name="${IMAGE_NAME}" \
        size="${SNAPSHOT_SIZE}" \
        zone="${ZONE}" \
        -o json
    )"
    SNAP_ID="$(echo "${IMPORT_JSON}" | jq -r '.id')"

    if [[ -z "${SNAP_ID}" || "${SNAP_ID}" == "null" ]]; then
      echo "ERREUR: import du snapshot échoué dans la zone ${ZONE}" >&2
      echo "${IMPORT_JSON}" >&2
      exit 1
    fi
    log "Snapshot importé dans ${ZONE}: ${SNAP_ID}"
    scw block snapshot wait "${SNAP_ID}" zone="${ZONE}" terminal-status=available
  fi

  # On supprime les anciennes images avec le nom stable pour éviter les conflits
  OLD_IMAGES="$(scw instance image list zone="${ZONE}" name="${STABLE_IMAGE_NAME}" -o json | jq -r '.[].id')"
  for OLD in ${OLD_IMAGES}; do
    log "Suppression de l'ancienne image ${OLD} (${STABLE_IMAGE_NAME}) dans ${ZONE}..."
    scw instance image delete "${OLD}" zone="${ZONE}" -f || true
  done

  log "Création de l'image stable '${STABLE_IMAGE_NAME}' dans ${ZONE}..."
  NEW_IMAGE_JSON="$(
    scw instance image create \
      snapshot-id="${SNAP_ID}" \
      name="${STABLE_IMAGE_NAME}" \
      arch="x86_64" \
      zone="${ZONE}" \
      -o json
  )"
  NEW_IMAGE_ID="$(echo "${NEW_IMAGE_JSON}" | jq -r '.id')"

  if [[ -z "${NEW_IMAGE_ID}" || "${NEW_IMAGE_ID}" == "null" ]]; then
    echo "ERREUR: échec de la création de l'image stable dans ${ZONE}" >&2
    echo "${NEW_IMAGE_JSON}" >&2
    exit 1
  fi
  log "Image stable créée dans ${ZONE}: ${NEW_IMAGE_ID}"
done

log "✅ Toutes les zones disposent de l'image stable '${STABLE_IMAGE_NAME}'."
