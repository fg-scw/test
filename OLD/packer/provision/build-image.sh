#!/bin/bash
set -e

# Variables
TALOS_VERSION="${TALOS_VERSION:-v1.11.5}"
WORK_DIR="/tmp/talos-build"
SCHEMATIC_FILE="/tmp/schematic.yaml"
TARGET_DEVICE="/dev/vda"

echo "==> Configuration"
echo "    Talos Version: ${TALOS_VERSION}"
echo "    Work Directory: ${WORK_DIR}"
echo "    Target Device: ${TARGET_DEVICE}"

# Créer le répertoire de travail
mkdir -p "${WORK_DIR}"
cd "${WORK_DIR}"

echo ""
echo "==> Étape 1: Génération du schematic ID via Image Factory"

# Lire le schematic YAML
if [ ! -f "${SCHEMATIC_FILE}" ]; then
    echo "ERREUR: Fichier schematic non trouvé: ${SCHEMATIC_FILE}"
    exit 1
fi

# Soumettre le schematic à l'Image Factory
echo "    Soumission du schematic..."
SCHEMATIC_RESPONSE=$(curl -sSL -X POST \
    --data-binary @"${SCHEMATIC_FILE}" \
    -H "Content-Type: application/yaml" \
    https://factory.talos.dev/schematics)

# Extraire le schematic ID
SCHEMATIC_ID=$(echo "${SCHEMATIC_RESPONSE}" | jq -r '.id')

if [ -z "${SCHEMATIC_ID}" ] || [ "${SCHEMATIC_ID}" = "null" ]; then
    echo "ERREUR: Impossible d'obtenir le schematic ID"
    echo "Réponse: ${SCHEMATIC_RESPONSE}"
    exit 1
fi

echo "    Schematic ID: ${SCHEMATIC_ID}"

echo ""
echo "==> Étape 2: Téléchargement de l'image Talos depuis Image Factory"

# URL de l'image Talos pour Scaleway (format raw compressé)
IMAGE_URL="https://factory.talos.dev/image/${SCHEMATIC_ID}/${TALOS_VERSION}/scaleway-amd64.raw.zst"
IMAGE_FILE="${WORK_DIR}/scaleway-amd64.raw.zst"

echo "    URL: ${IMAGE_URL}"
echo "    Téléchargement..."

wget -q --show-progress "${IMAGE_URL}" -O "${IMAGE_FILE}"

if [ ! -f "${IMAGE_FILE}" ]; then
    echo "ERREUR: Téléchargement de l'image échoué"
    exit 1
fi

echo "    Téléchargement terminé: $(du -h ${IMAGE_FILE} | cut -f1)"

echo ""
echo "==> Étape 3: Décompression de l'image"

RAW_FILE="${WORK_DIR}/scaleway-amd64.raw"
echo "    Décompression avec zstd..."
zstd -d -f "${IMAGE_FILE}" -o "${RAW_FILE}"

if [ ! -f "${RAW_FILE}" ]; then
    echo "ERREUR: Décompression échouée"
    exit 1
fi

echo "    Décompression terminée: $(du -h ${RAW_FILE} | cut -f1)"

echo ""
echo "==> Étape 4: Préparation du disque cible"

# Vérifier que le device existe
if [ ! -b "${TARGET_DEVICE}" ]; then
    echo "ERREUR: Device ${TARGET_DEVICE} non trouvé"
    exit 1
fi

echo "    Nettoyage de la table de partitions..."
sgdisk --zap-all "${TARGET_DEVICE}" 2>/dev/null || true
sync

echo ""
echo "==> Étape 5: Écriture de l'image sur le disque"

echo "    Écriture avec dd..."
echo "    ATTENTION: Cela va écraser complètement ${TARGET_DEVICE}"
sleep 2

# Écrire l'image raw sur le disque
dd if="${RAW_FILE}" of="${TARGET_DEVICE}" bs=8M status=progress oflag=direct

echo ""
echo "==> Étape 6: Synchronisation et vérification"

echo "    Synchronisation des écritures..."
sync

echo "    Vérification des partitions..."
partprobe "${TARGET_DEVICE}" 2>/dev/null || true
sleep 2

echo "    Affichage de la table de partitions:"
parted -s "${TARGET_DEVICE}" print || true

echo ""
echo "==> Étape 7: Nettoyage"
cd /
rm -rf "${WORK_DIR}"

echo ""
echo "==> ✅ Image Talos ${TALOS_VERSION} écrite avec succès sur ${TARGET_DEVICE}"
echo "    Schematic ID: ${SCHEMATIC_ID}"
echo ""
echo "Le snapshot Packer va maintenant être créé..."
