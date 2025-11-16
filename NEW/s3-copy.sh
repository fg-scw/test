#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# copy-via-s3.sh
#
# Objectif :
# - Récupérer automatiquement la dernière image Talos créée par Packer
#   via manifest.json
# - Exporter le snapshot root vers S3
# - Importer ce snapshot dans chaque AZ du region (fr-par-1/2/3)
# - Créer dans chaque AZ une image STABLE sans timestamp :
#     talos-scaleway-<talos_version>
#
# Usage :
#   ./copy-via-s3.sh <bucket-name>
#
# Ex :
#   ./copy-via-s3.sh talos-fgz
#
# Pré-requis :
# - scaleway-cli installé et configuré (SCW_ACCESS_KEY, SCW_SECRET_KEY, projet)
# - jq installé
# - manifest.json généré par Packer dans ce répertoire
# ---------------------------------------------------------------------------

if ! command -v scw >/dev/null 2>&1; then
  echo "Erreur: 'scw' (Scaleway CLI) n'est pas installé dans le PATH." >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "Erreur: 'jq' n'est pas installé dans le PATH." >&2
  exit 1
fi

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <s3-bucket-name>" >&2
  exit 1
fi

BUCKET_NAME="$1"
REGION="${REGION:-fr-par}"
TARGET_ZONES="${TARGET_ZONES:-fr-par-1 fr-par-2 fr-par-3}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

MANIFEST_FILE="manifest.json"
if [[ ! -f "${MANIFEST_FILE}" ]]; then
  echo "Erreur: ${MANIFEST_FILE} introuvable dans ${SCRIPT_DIR}" >&2
  exit 1
fi

echo "==> Lecture de ${MANIFEST_FILE}"
# On prend le premier (ou dernier) build du manifest
ARTIFACT_ID="$(jq -r '.builds[-1].artifact_id' "${MANIFEST_FILE}")"
if [[ -z "${ARTIFACT_ID}" || "${ARTIFACT_ID}" == "null" ]]; then
  echo "Erreur: impossible de récupérer artifact_id depuis manifest.json" >&2
  exit 1
fi

SRC_ZONE="${ARTIFACT_ID%%:*}"
SRC_IMAGE_ID="${ARTIFACT_ID##*:}"

echo "==> Image Packer trouvée :"
echo "    Zone source : ${SRC_ZONE}"
echo "    Image ID    : ${SRC_IMAGE_ID}"

echo "==> Récupération des métadonnées de l'image"
IMAGE_JSON="$(scw instance image get "${SRC_IMAGE_ID}" zone="${SRC_ZONE}" -o json)"

IMAGE_NAME="$(echo "${IMAGE_JSON}" | jq -r '.name')"

if [[ -z "${IMAGE_NAME}" || "${IMAGE_NAME}" == "null" ]]; then
  echo "Erreur: impossible de récupérer le nom de l'image depuis 'scw instance image get'" >&2
  exit 1
fi

echo "    Image name  : ${IMAGE_NAME}"

# On essaie de trouver le snapshot root dans les champs possibles
ROOT_SNAPSHOT_ID="$(
  echo "${IMAGE_JSON}" | jq -r '
    .root_volume.id // .from_root_volume.id // .snapshot_id // empty
  '
)"

if [[ -z "${ROOT_SNAPSHOT_ID}" ]]; then
  echo "Erreur: impossible de récupérer le snapshot root depuis l'image." >&2
  echo "JSON de l'image:"
  echo "${IMAGE_JSON}" | jq .
  exit 1
fi

echo "    Root snapshot ID : ${ROOT_SNAPSHOT_ID}"

echo "==> Lecture des infos du snapshot"
SNAP_JSON="$(scw block snapshot get "${ROOT_SNAPSHOT_ID}" zone="${SRC_ZONE}" -o json)"
SNAP_SIZE="$(echo "${SNAP_JSON}" | jq -r '.size')"

if [[ -z "${SNAP_SIZE}" || "${SNAP_SIZE}" == "null" ]]; then
  echo "Erreur: impossible de récupérer la taille du snapshot." >&2
  echo "${SNAP_JSON}" | jq .
  exit 1
fi

echo "    Snapshot size    : ${SNAP_SIZE}"

# On dérive un nom STABLE sans timestamp en coupant le suffixe après le dernier '-'
# Exemple: talos-scaleway-v1.11.5-20251115220248 -> talos-scaleway-v1.11.5
STABLE_IMAGE_NAME="${IMAGE_NAME%-*}"

if [[ "${STABLE_IMAGE_NAME}" == "${IMAGE_NAME}" ]]; then
  echo "Attention: le nom d'image ne semble pas contenir de timestamp."
  echo "On utilisera quand même '${STABLE_IMAGE_NAME}' comme nom stable."
fi

OBJECT_KEY="${STABLE_IMAGE_NAME}.qcow2"

echo
echo "============================================================"
echo "Export du snapshot root vers S3"
echo "  Bucket : ${BUCKET_NAME}"
echo "  Key    : ${OBJECT_KEY}"
echo "============================================================"

scw block snapshot export-to-object-storage \
  snapshot-id="${ROOT_SNAPSHOT_ID}" \
  bucket="${BUCKET_NAME}" \
  key="${OBJECT_KEY}" \
  zone="${SRC_ZONE}" -o json >/tmp/talos-export-task.json

echo "=> Export déclenché. Attente pour laisser l'export se terminer..."
# Approche simple mais fiable: on laisse du temps pour la copie réelle
sleep 120

echo
echo "============================================================"
echo "Import du snapshot et création des images STABLE par AZ"
echo "  Nom stable de l'image : ${STABLE_IMAGE_NAME}"
echo "  Zones cibles          : ${TARGET_ZONES}"
echo "============================================================"

for ZONE in ${TARGET_ZONES}; do
  echo
  echo "==> Zone ${ZONE}"

  echo "  - Import du snapshot depuis S3..."
  IMPORT_JSON="$(
    scw block snapshot import-from-object-storage \
      bucket="${BUCKET_NAME}" \
      key="${OBJECT_KEY}" \
      name="${STABLE_IMAGE_NAME}-snapshot" \
      size="${SNAP_SIZE}" \
      zone="${ZONE}" -o json
  )"

  NEW_SNAPSHOT_ID="$(echo "${IMPORT_JSON}" | jq -r '.id')"

  if [[ -z "${NEW_SNAPSHOT_ID}" || "${NEW_SNAPSHOT_ID}" == "null" ]]; then
    echo "    !! Erreur: impossible de récupérer l'ID du snapshot importé pour ${ZONE}" >&2
    echo "${IMPORT_JSON}" | jq .
    exit 1
  fi

  echo "    Snapshot importé : ${NEW_SNAPSHOT_ID}"

  echo "  - Suppression des anciennes images '${STABLE_IMAGE_NAME}' (s'il y en a)..."
  # On liste les images existantes du même nom dans la zone et on les supprime
  EXISTING_IMAGES="$(
    scw instance image list "name=${STABLE_IMAGE_NAME}" zone="${ZONE}" -o json | jq -r '.[].id'
  )" || true

  if [[ -n "${EXISTING_IMAGES}" ]]; then
    while read -r OLD_ID; do
      [[ -z "${OLD_ID}" ]] && continue
      echo "    -> Suppression image ${OLD_ID}"
      scw instance image delete "${OLD_ID}" zone="${ZONE}" >/dev/null 2>&1 || true
    done <<< "${EXISTING_IMAGES}"
  else
    echo "    Aucune image existante à supprimer."
  fi

  echo "  - Création de l'image stable '${STABLE_IMAGE_NAME}' dans ${ZONE}..."
  CREATE_JSON="$(
    scw instance image create \
      snapshot-id="${NEW_SNAPSHOT_ID}" \
      name="${STABLE_IMAGE_NAME}" \
      arch="x86_64" \
      zone="${ZONE}" -o json
  )"

  NEW_IMAGE_ID="$(echo "${CREATE_JSON}" | jq -r '.id')"

  if [[ -z "${NEW_IMAGE_ID}" || "${NEW_IMAGE_ID}" == "null" ]]; then
    echo "    !! Erreur: impossible de récupérer l'ID de la nouvelle image pour ${ZONE}" >&2
    echo "${CREATE_JSON}" | jq .
    exit 1
  fi

  echo "    Image stable créée dans ${ZONE} : ${NEW_IMAGE_ID}"
done

echo
echo "✅ Synchronisation des images Talos terminée."
echo "   Nom d'image STABLE à utiliser dans Terraform : ${STABLE_IMAGE_NAME}"
