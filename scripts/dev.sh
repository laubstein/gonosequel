#!/usr/bin/env bash
# Brings up a full local dev environment with a single command:
#   - a MongoDB to point at (reuses one already listening on MONGO_PORT,
#     otherwise starts one in Docker)
#   - the Go API server, in the background
#   - the Vite dev server in the foreground — browse here, not the Go
#     port: Vite's own proxy forwards /api to the Go server (see
#     web/vite.config.ts) and, unlike the Go server's --dev-proxy flag,
#     supports the WebSocket upgrade hot module reload needs.
#
# Invoked via `make dev`; see the Makefile for the MONGO_PORT / HTTP_PORT /
# VITE_PORT variables it honors.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

MONGO_PORT="${MONGO_PORT:-27017}"
HTTP_PORT="${HTTP_PORT:-8081}"
VITE_PORT="${VITE_PORT:-5173}"
MONGO_CONTAINER="gonosequel-dev-mongo"

mongo_reachable() {
  timeout 2 bash -c "exec 3<>/dev/tcp/127.0.0.1/${MONGO_PORT}" 2>/dev/null
}

if mongo_reachable; then
  echo "==> Using MongoDB already listening on localhost:${MONGO_PORT}"
else
  if ! command -v docker >/dev/null 2>&1; then
    echo "error: no MongoDB found on localhost:${MONGO_PORT}, and Docker is not installed to start one." >&2
    echo "Start a MongoDB yourself, or install Docker and retry." >&2
    exit 1
  fi

  if docker ps -a --format '{{.Names}}' | grep -qx "$MONGO_CONTAINER"; then
    echo "==> Starting existing dev MongoDB container ($MONGO_CONTAINER)"
    docker start "$MONGO_CONTAINER" >/dev/null
  else
    echo "==> No MongoDB on localhost:${MONGO_PORT}; starting one in Docker ($MONGO_CONTAINER)"
    docker run -d --name "$MONGO_CONTAINER" -p "${MONGO_PORT}:27017" mongo:8 >/dev/null
  fi

  printf "==> Waiting for MongoDB to accept connections"
  ready=0
  for _ in $(seq 1 30); do
    if mongo_reachable; then
      ready=1
      break
    fi
    printf "."
    sleep 1
  done
  echo
  if [ "$ready" -ne 1 ]; then
    echo "error: MongoDB container did not become ready in time; check 'docker logs $MONGO_CONTAINER'" >&2
    exit 1
  fi
fi

# go:embed requires web/dist to exist at compile time even though this
# dev workflow never serves its contents (the browser talks to Vite, not
# the Go server's asset route). On a fresh clone nothing has built it
# yet, so make sure it's at least present.
if [ ! -f web/dist/index.html ]; then
  echo "==> Creating placeholder web/dist (go:embed needs it to exist; dev mode serves the frontend from Vite, not this)"
  mkdir -p web/dist
  echo '<!doctype html><title>gonosequel (dev placeholder)</title>' > web/dist/index.html
fi

if [ ! -d web/node_modules ]; then
  echo "==> Installing frontend dependencies"
  (cd web && npm ci)
fi

# Unlike the frontend (served live by Vite above) and the API server
# itself (rebuilt on every run), /doc is served from docs/.vitepress/dist
# via go:embed — baked into the binary at compile time, with no live
# reload. Rebuilding it here on every `make dev` run is what makes doc
# edits actually show up; skipping this step (as this script used to)
# left /doc silently serving whatever was last built by hand, however
# stale.
if [ ! -d docs/node_modules ]; then
  echo "==> Installing docs dependencies"
  (cd docs && npm ci)
fi
echo "==> Building docs (embedded into the API server's /doc route)"
(cd docs && npm run docs:build)

echo "==> Building gonosequel"
API_BIN="$(mktemp -d)/gonosequel-dev"
go build -o "$API_BIN" .

echo "==> Starting gonosequel API on :${HTTP_PORT}"
# A real binary, not `go run .`: go run's own process never execs into
# the compiled binary, it runs it as a child, so killing go run's PID
# leaves that child running behind — confirmed while testing this script.
"$API_BIN" --url "mongodb://localhost:${MONGO_PORT}" --http-port "$HTTP_PORT" &
API_PID=$!

# Both child processes run in the background and the script waits on
# them explicitly (see the final `wait`) rather than running the last
# one in the foreground: bash only runs traps between commands, and while
# it's blocked on a foreground child it defers even a caught signal like
# SIGINT until that child exits on its own — which, for a dev server,
# never happens. `wait` is interruptible, so Ctrl+C reaches this trap
# immediately instead of only after the process is killed some other way.
cleanup() {
  echo
  echo "==> Stopping dev servers"
  kill "$API_PID" "${VITE_PID:-}" 2>/dev/null || true
  wait "$API_PID" "${VITE_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

printf "==> Waiting for the API to come up"
api_ready=0
for _ in $(seq 1 30); do
  if curl -sf "http://localhost:${HTTP_PORT}/api/info" >/dev/null 2>&1; then
    api_ready=1
    break
  fi
  printf "."
  sleep 1
done
echo
if [ "$api_ready" -ne 1 ]; then
  echo "error: API server did not come up in time" >&2
  exit 1
fi

echo "==> Starting Vite dev server — open http://localhost:${VITE_PORT} in your browser"
# The vite binary directly, not `npm run dev`: an extra npm process in
# between would receive our kill signal instead of vite itself, since
# older npm versions don't reliably forward signals to the child they
# spawn.
(cd web && exec env VITE_API_PORT="$HTTP_PORT" ./node_modules/.bin/vite --port "$VITE_PORT" --strictPort) &
VITE_PID=$!

wait -n "$API_PID" "$VITE_PID"
