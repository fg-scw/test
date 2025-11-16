#!/bin/bash
set -euo pipefail

# ============================
#  Variables de configuration
# ============================
TALOS_VERSION="${TALOS_VERSION:-v1.11.5}"
WORK_DIR="${WORK_DIR:-/tmp/talos-build}"
SCHEMATIC_FILE="${SCHEMATIC_FILE:-/tmp/schematic.yaml}"

# ============================
#  Auto-détection du disque
# ============================
# Si TARGET_DEVICE est déjà défini dans l'environnement, on le respecte.
if [ -z "${TARGET_DEVICE:-}" ]; then
  # On prend le premier "disk" retourné par lsblk (cas classique Scaleway : /dev/sda)
  DETECTED_DISK="$(lsblk -dn -o NAME,TYPE | awk '$2=="disk"{print $1; exit}')"
  if [ -n "${DETECTED_DISK}" ]; then
    TARGET_DEVICE="/dev/${DETECTED_DISK}"
  else
    TARGET_DEVICE=""
  fi
fi

if [ -z "${TARGET_DEVICE}" ] || [ ! -b "${TARGET_DEVICE}" ]; then
  echo "ERREUR: Device ${TARGET_DEVICE:-<non défini>} non trouvé" >&2
  echo "lsblk :"
  lsblk || true
  exit 1
fi

# ============================
#  Vérification des binaires
# ============================
REQUIRED_CMDS=(curl jq wget zstd lsblk dd)
for cmd in "${REQUIRED_CMDS[@]}"; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "ERREUR: Commande requise manquante: ${cmd}" >&2
    exit 1
  fi
done

echo "==> Configuration"
echo "    Talos Version: ${TALOS_VERSION}"
echo "    Work Directory: ${WORK_DIR}"
echo "    Target Device: ${TARGET_DEVICE}"
echo "    Schematic: ${SCHEMATIC_FILE}"

# ============================
#  Étape 0: Préparation du répertoire
# ============================
echo ""
echo "==> Étape 0: Préparation du répertoire de travail"

rm -rf "${WORK_DIR}"
mkdir -p "${WORK_DIR}"
cd "${WORK_DIR}"

# ============================
#  Étape 1: Schematic ID via Image Factory
# ============================
echo ""
echo "==> Étape 1: Génération du schematic ID via Image Factory"
echo "    Soumission du schematic..."

if [ ! -f "${SCHEMATIC_FILE}" ]; then
  echo "ERREUR: Fichier schematic introuvable: ${SCHEMATIC_FILE}" >&2
  exit 1
fi

SCHEMATIC_ID="$(
  curl -fsSL -X POST \
    --data-binary @"${SCHEMATIC_FILE}" \
    https://factory.talos.dev/schematics \
  | jq -r '.id'
)"

if [ -z "${SCHEMATIC_ID}" ] || [ "${SCHEMATIC_ID}" = "null" ]; then
  echo "ERREUR: Impossible de récupérer le Schematic ID depuis Image Factory" >&2
  exit 1
fi

echo "    Schematic ID: ${SCHEMATIC_ID}"

# ============================
#  Étape 2: Téléchargement de l'image
# ============================
echo ""
echo "==> Étape 2: Téléchargement de l'image Talos depuis Image Factory"

IMAGE_URL="https://factory.talos.dev/image/${SCHEMATIC_ID}/${TALOS_VERSION}/scaleway-amd64.raw.zst"
RAW_ZST="${WORK_DIR}/scaleway-amd64.raw.zst"
RAW_IMG="${WORK_DIR}/scaleway-amd64.raw"

echo "    URL: ${IMAGE_URL}"
echo "    Téléchargement..."
wget -O "${RAW_ZST}" "${IMAGE_URL}"

echo "    Téléchargement terminé: $(du -h "${RAW_ZST}" | awk '{print $1}')"

# ============================
#  Étape 3: Décompression
# ============================
echo ""
echo "==> Étape 3: Décompression de l'image"
echo "    Décompression avec zstd..."

zstd -d -f --rm "${RAW_ZST}" -o "${RAW_IMG}"

echo "    Décompression terminée: $(du -h "${RAW_IMG}" | awk '{print $1}')"

# ============================
#  Étape 4: Préparation du disque cible
# ============================
echo ""
echo "==> Étape 4: Préparation du disque cible"

if [ ! -b "${TARGET_DEVICE}" ]; then
  echo "ERREUR: Device ${TARGET_DEVICE} non trouvé" >&2
  lsblk || true
  exit 1
fi

echo "    Informations sur le disque:"
lsblk "${TARGET_DEVICE}" || true

echo "    Nettoyage des premiers méga-octets (pour éviter les anciennes signatures)..."
dd if=/dev/zero of="${TARGET_DEVICE}" bs=1M count=4 conv=fsync || true

# ============================
#  Étape 5: Écriture de l'image Talos
# ============================
echo ""
echo "==> Étape 5: Écriture de l'image Talos sur ${TARGET_DEVICE}"

# IMPORTANT :
# - conv=fsync force le flush des données avant le retour de dd.
# - Après cet appel, on considère que l'OS en cours est "sacrifié"
#   (le disque root vient d'être remplacé), donc aucune commande externe
#   ne doit être appelée ensuite.
dd if="${RAW_IMG}" of="${TARGET_DEVICE}" bs=4M status=progress conv=fsync

# ============================
#  Étape 6: Fin propre (aucune commande externe)
# ============================
echo ""
echo "==> Étape 6: Fin de l'écriture"
echo "    L'image Talos ${TALOS_VERSION} a été écrite sur ${TARGET_DEVICE}."
echo "    Schematic ID: ${SCHEMATIC_ID}"
echo ""
echo "    Le système de fichiers d'origine vient d'être remplacé."
echo "    Aucune opération supplémentaire n'est effectuée pour éviter des erreurs."
echo ""
echo "==> ✅ Fin du script de provisioning (exit 0)"

exit 0
