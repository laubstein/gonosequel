# CLI reference

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--url` | `ME_URL` | — | Full MongoDB connection URL. Takes priority over `--host`/`--port`/`--user`/`--pass`/`--db`. |
| `--host` | `ME_HOST` | `localhost` | MongoDB host, used when `--url` isn't given. |
| `--port` | `ME_MONGO_PORT` | `27017` | MongoDB port. |
| `--user` / `--pass` | `ME_USER` / `ME_PASS` | — | MongoDB credentials. |
| `--db` | `ME_DB` | — | Default database. |
| `--bookmark` | `ME_BOOKMARK` | — | Load the connection from a saved bookmark instead of the flags above. Takes priority over `--url`. See [Connecting](/connecting). |
| `--bind` | `ME_BIND` | `127.0.0.1` | Address the HTTP server binds to. |
| `--http-port` | `ME_HTTP_PORT` | `8081` | HTTP server port. |
| `--sessions` | `ME_SESSIONS` | `false` | Multi-session mode — see [Connecting](/connecting). |
| `--readonly` | `ME_READONLY` | `false` | Reject every non-GET API request with 403, enforced server-side. See [Read-only mode](/readonly-mode). |
| `--auth-user` / `--auth-pass` | `ME_AUTH_USER` / `ME_AUTH_PASS` | — | Basic auth in front of the web UI itself. |
| `--dev-proxy` | `ME_DEV_PROXY` | — | Reverse-proxy non-API requests to this URL instead of serving the embedded frontend. Used internally by `make dev`; prefer browsing Vite's own URL directly in development, since this path doesn't support the WebSocket upgrade hot module reload needs. |

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
