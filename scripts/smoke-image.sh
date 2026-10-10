#!/usr/bin/env bash
# Runs a built image against a real Postgres container and checks it works end to end:
# it starts and reports itself healthy, reports the expected version, serves the frontend,
# and a request goes through the API into the database and back.
#
#   scripts/smoke-image.sh <image> [expected-version]
#
# Needs Docker. SMOKE_PORT (default 18080) is the host port used for the app.
set -euo pipefail

image=${1:?usage: smoke-image.sh <image> [expected-version]}
want=${2:-}
port=${SMOKE_PORT:-18080}
p=smoke-$$

cleanup() {
  docker rm -f "$p-app" "$p-db" >/dev/null 2>&1 || true
  docker network rm "$p" >/dev/null 2>&1 || true
}
trap cleanup EXIT
fail() { echo "FAIL: $*" >&2; docker logs "$p-app" 2>&1 | tail -20 >&2 || true; exit 1; }
ok() { echo "ok: $*"; }

retry() { # retry <seconds> <command...>: run the command until it succeeds or the time is up
  local limit=$1; shift
  for _ in $(seq 1 "$limit"); do "$@" >/dev/null 2>&1 && return 0; sleep 1; done
  return 1
}

docker network create "$p" >/dev/null
docker run -d --name "$p-db" --network "$p" -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=ares_bib_logger \
  --health-cmd "pg_isready -U postgres -d ares_bib_logger" --health-interval 2s --health-retries 30 postgres:17-alpine >/dev/null
db_ready() { [ "$(docker inspect -f '{{.State.Health.Status}}' "$p-db")" = healthy ]; }
retry 60 db_ready || fail "Postgres did not become ready"

docker run -d --name "$p-app" --network "$p" -p "$port:8080" \
  -e DB_HOST="$p-db" -e DB_USER=postgres -e DB_PASSWORD=pw -e DB_NAME=ares_bib_logger \
  -e MQTT_ENABLED=false "$image" >/dev/null
url=http://127.0.0.1:$port
retry 30 curl -fsS "$url/health" || fail "the app did not answer /health (migrations may have failed)"
ok "the app is up and has migrated the database"

# The image has no shell or curl: its own healthcheck command is what Docker runs.
docker exec "$p-app" /app/server healthcheck || fail "the healthcheck command failed inside the container"
ok "the healthcheck command works"

if [ -n "$want" ]; then
  got=$(curl -fsS "$url/health")
  case "$got" in *"\"version\":\"$want\""*) ok "version $want" ;; *) fail "expected version $want, got $got" ;; esac
fi

curl -fsS "$url/" | grep -q 'id="root"' || fail "the frontend was not served"
ok "the frontend is served"

# Through the whole path: create an event over the API and read it back from Postgres.
created=$(curl -fsS -X POST "$url/api/events" -H 'Content-Type: application/json' -d '{"name":"Smoke"}') \
  || fail "could not create an event"
curl -fsS "$url/api/events" | grep -q '"Smoke"' || fail "the created event ($created) was not listed"
ok "an event created over the API is read back from Postgres"

echo "image $image passed"
