#!/bin/bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <VLLM_BASE_URL>"
  echo "Exemple: $0 http://localhost:8000/v1"
  exit 1
fi

BASE_URL="$1"

echo "Test de l'API vLLM sur ${BASE_URL}"
echo ""

echo "1. Liste des modèles..."
curl -sS "${BASE_URL}/models" | jq '.' || true
echo ""

echo "2. Test completions (OpenAI compatible)..."
curl -sS "${BASE_URL}/completions"   -H "Content-Type: application/json"   -d '{
    "model": "mistralai/Mistral-7B-Instruct-v0.2",
    "prompt": "Explique ce qu\u0027est Ray et pourquoi on l\u0027utilise avec vLLM.",
    "max_tokens": 128,
    "temperature": 0.7
  }' | jq '.' || true
echo ""

echo "3. Test chat completions (OpenAI compatible)..."
curl -sS "${BASE_URL}/chat/completions"   -H "Content-Type: application/json"   -d '{
    "model": "mistralai/Mistral-7B-Instruct-v0.2",
    "messages": [
      {"role": "system", "content": "You are a Kubernetes and Ray expert."},
      {"role": "user", "content": "Donne un exemple d\u0027architecture Ray + vLLM sur un cluster multi-AZ."}
    ],
    "max_tokens": 256,
    "temperature": 0.7
  }' | jq '.' || true
