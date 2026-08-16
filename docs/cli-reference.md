# CLI reference

Every env var below is read as `GNS_<name>` first; if unset, `ME_<name>` (mongo-express's own
naming — this project's earlier working name) is used as a fallback, so an existing
mongo-express deployment can switch images without also rewriting its environment. A value
containing `$VAR` or `${VAR}` is expanded against the rest of the environment before use (the
same substitution a shell does), so e.g. `GNS_URL=$MONGO_URL` resolves through to whatever
`MONGO_URL` is set to — handy when a value is already wired up by the surrounding deployment (a
docker-compose service link, a Kubernetes downstream env var, etc).

`--host`/`--port`/`--user`/`--pass` get one more fallback tier, consulted only when `--driver`
is `mongodb`: `MONGODB_HOST`/`MONGODB_PORT`/`MONGODB_USERNAME`/`MONGODB_PASSWORD` — the
convention used by official MongoDB Docker images and common Helm chart deployments, so
gonosequel can pick up connection details already present in the environment without
duplicating them under a `GNS_`/`ME_` name. Not consulted for `redis`/`valkey`, where it
wouldn't mean anything.

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--driver` | `GNS_DRIVER` (`ME_DRIVER`) | `mongodb` | Backend to connect to: `mongodb`, `redis`, or `valkey` (the latter two are wire-compatible and route to the same driver). See [Redis & Valkey](/features/redis-valkey). |
| `--url` | `GNS_URL` (`ME_URL`) | — | Full connection URL (`mongodb://...` or `redis://...`). Takes priority over `--host`/`--port`/`--user`/`--pass`/`--db`. |
| `--host` | `GNS_HOST` (`ME_HOST`, then `MONGODB_HOST` for MongoDB) | `localhost` | Backend host, used when `--url` isn't given. |
| `--port` | `GNS_PORT` (`ME_PORT`, then `MONGODB_PORT` for MongoDB) | `27017` (MongoDB) / `6379` (Redis/Valkey) | Backend port; the fallback default depends on `--driver`. |
| `--user` / `--pass` | `GNS_USER` / `GNS_PASS` (`ME_USER` / `ME_PASS`, then `MONGODB_USERNAME` / `MONGODB_PASSWORD` for MongoDB) | — | Backend credentials. |
| `--db` | `GNS_DB` (`ME_DB`) | — | Default database (MongoDB database name, or a Redis/Valkey numbered database 0-15). |
| `--bookmark` | `GNS_BOOKMARK` (`ME_BOOKMARK`) | — | Load the connection from a saved bookmark instead of the flags above. Takes priority over `--url`. See [Connecting](/connecting). |
| `--bind` | `GNS_BIND` (`ME_BIND`) | `127.0.0.1` | Address the HTTP server binds to. |
| `--http-port` | `GNS_HTTP_PORT` (`ME_HTTP_PORT`) | `8081` | HTTP server port. |
| `--sessions` | `GNS_SESSIONS` (`ME_SESSIONS`) | `false` | Multi-session mode — see [Connecting](/connecting). |
| `--readonly` | `GNS_READONLY` (`ME_READONLY`) | `false` | Reject every non-GET API request with 403, enforced server-side. See [Read-only mode](/readonly-mode). |
| `--auth-user` / `--auth-pass` | `GNS_AUTH_USER` / `GNS_AUTH_PASS` (`ME_AUTH_USER` / `ME_AUTH_PASS`) | — | Basic auth in front of the web UI itself. |
| `--dev-proxy` | `GNS_DEV_PROXY` (`ME_DEV_PROXY`) | — | Reverse-proxy non-API requests to this URL instead of serving the embedded frontend. Used internally by `make dev`; prefer browsing Vite's own URL directly in development, since this path doesn't support the WebSocket upgrade hot module reload needs. |

Every flag also has a `--read-timeout` / `--write-timeout` pair (both default `30s`) for the
HTTP server's own timeouts, and standard Go `flag` package behavior applies: `-flag=value`,
`-flag value`, and `--flag` are all accepted, and `--help` prints this same list from the
binary itself.

## Example: read-only, bound to all interfaces, behind basic auth

```bash
./gonosequel \
  --url mongodb://localhost:27017 \
  --bind 0.0.0.0 --http-port 8081 \
  --readonly \
  --auth-user admin --auth-pass "$ADMIN_PASSWORD"
```
