#!/usr/bin/env bash
# Deploy one release to this VM, and roll back if it is not healthy.
#
#   ./deploy.sh v1.2.0
#
# Pulls the images release.yml built for the tag, starts the stack, makes sure the Ollama
# models are present, then waits for /health/ready (LLM, Postgres and Qdrant
# all answering). If the new release is not ready in time, the previous tag is
# started again and the script exits 1. The deployed tag is kept in
# .deployed_tag.
set -euo pipefail
cd "$(dirname "$0")"

tag=${1:?usage: deploy.sh <release tag, e.g. v1.2.0>}
if ! [[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "deploy: $tag is not a release tag like v1.2.0" >&2
  exit 2
fi
if [[ ! -f .env ]]; then
  echo "deploy: .env is missing: cp prod.env.example .env and fill it in" >&2
  exit 2
fi

# Read only the variables this script needs; never echo the file.
env_value() { sed -n "s/^$1=//p" .env | tail -n1; }
domain=$(env_value DOMAIN)
model=$(env_value OLLAMA_MODEL)
model=${model:-llama3.2:3b}
embed_model=nomic-embed-text
ready_timeout=${READY_TIMEOUT:-300}

# Git tags are v1.2.0; release.yml publishes the images as 1.2.0.
compose() { IMAGE_TAG=${1#v} docker compose --env-file .env -f docker-compose.yml "${@:2}"; }

ensure_model() {
  local t=$1 m=$2
  if ! compose "$t" exec -T ollama ollama show "$m" >/dev/null 2>&1; then
    echo "deploy: pulling Ollama model $m (first time only)"
    compose "$t" exec -T ollama ollama pull "$m" 2>&1 | tail -n 1 # drop the progress bars
  fi
}

# Probe through Caddy on this host. -k: the first certificate may still be
# issuing; this checks the app, not TLS.
wait_ready() {
  local deadline=$((SECONDS + ready_timeout))
  while ((SECONDS < deadline)); do
    if curl -fsS -k -m 20 --resolve "$domain:443:127.0.0.1" "https://$domain/health/ready" >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  return 1
}

start() {
  local t=$1
  # PULL=0 rehearses with locally built images; production always pulls.
  if [[ ${PULL:-1} != 0 ]]; then
    compose "$t" pull --quiet
  fi
  compose "$t" up -d --remove-orphans
  ensure_model "$t" "$model"
  ensure_model "$t" "$embed_model"
}

prev=$(cat .deployed_tag 2>/dev/null || true)
echo "deploy: $prev -> $tag"
start "$tag"

if wait_ready; then
  echo "$tag" > .deployed_tag
  echo "deploy: $tag is ready at https://$domain"
  compose "$tag" exec -T ollama ollama list | sed 's/^/deploy: model /'
  docker image prune -f >/dev/null
  exit 0
fi

echo "deploy: $tag did not become ready within ${ready_timeout}s" >&2
compose "$tag" logs --tail 40 api >&2 || true
curl -sS -k -m 20 --resolve "$domain:443:127.0.0.1" "https://$domain/health/ready" >&2 || true

if [[ -n $prev && $prev != "$tag" ]]; then
  echo "deploy: rolling back to $prev" >&2
  start "$prev"
  if wait_ready; then
    echo "deploy: rolled back to $prev" >&2
  else
    echo "deploy: rollback to $prev is not ready either: check the VM" >&2
  fi
fi
exit 1
