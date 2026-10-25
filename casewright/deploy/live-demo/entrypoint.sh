#!/bin/sh
# Starts the API and nginx, creates the shared demo analyst, and resets the
# demo every DEMO_RESET_MINUTES by restarting the API (storage is in memory).
set -eu

DEMO_USER="${DEMO_USER:-demo}"
DEMO_PASSWORD="${DEMO_PASSWORD:-casewright-demo}"
RESET_MINUTES="${DEMO_RESET_MINUTES:-360}"

# Admin credentials are random per start and never shown: visitors use the
# analyst account and cannot manage users. Anything a visitor changes,
# including the demo password, is undone by the next reset.
export CASEWRIGHT_ADMIN_USER=admin
export CASEWRIGHT_ADMIN_PASSWORD="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
export CASEWRIGHT_API_KEYS="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
export LLM_PROVIDER=none
export CONDUCTOR_PORT=8080
export CASEWRIGHT_COOKIE_SECURE="${CASEWRIGHT_COOKIE_SECURE:-true}"  # Spaces serve HTTPS
export SEED_LOG_FILE=/demo/attack-scenario.log
export IOC_WATCHLIST_DIR=/demo/watchlists
export RESPONSE_WEBHOOK_URL=http://127.0.0.1:9000/contain
export SLACK_WEBHOOK_URL=http://127.0.0.1:9000/notify
unset POSTGRES_DSN

start_api() {
  /usr/local/bin/casewright --serve &
  API_PID=$!
  i=0
  until curl -fs http://127.0.0.1:8080/health >/dev/null; do
    i=$((i + 1)); [ "$i" -gt 60 ] && { echo "API did not start" >&2; exit 1; }
    sleep 1
  done
  curl -fs -o /dev/null -X POST http://127.0.0.1:8080/api/users \
    -H "X-API-Key: $CASEWRIGHT_API_KEYS" -H "Content-Type: application/json" \
    -d "{\"username\":\"$DEMO_USER\",\"display_name\":\"Demo analyst\",\"password\":\"$DEMO_PASSWORD\",\"role\":\"analyst\"}" \
    && echo "demo analyst '$DEMO_USER' ready"
}

nginx -g 'daemon off;' &
NGINX_PID=$!
trap 'kill $API_PID $NGINX_PID 2>/dev/null; exit 0' TERM INT

start_api
while true; do
  sleep "$((RESET_MINUTES * 60))" &
  wait $!
  echo "resetting the demo"
  kill "$API_PID" 2>/dev/null; wait "$API_PID" 2>/dev/null || true
  start_api
done
