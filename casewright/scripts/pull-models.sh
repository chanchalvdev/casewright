#!/usr/bin/env bash
set -euo pipefail

OLLAMA_URL="${OLLAMA_BASE_URL:-http://localhost:11434}"

echo "Waiting for Ollama to be ready at ${OLLAMA_URL}..."
until curl -sf "${OLLAMA_URL}/api/tags" > /dev/null 2>&1; do
  sleep 2
done
echo "Ollama is ready."

pull_model() {
  local model="$1"
  echo "Pulling ${model}..."
  curl -sf "${OLLAMA_URL}/api/pull" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"${model}\"}" \
    | tail -1
  echo "Done: ${model}"
}

pull_model "nomic-embed-text"
pull_model "llama3.2"

echo "All models ready."
