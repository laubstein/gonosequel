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

`--auth-user`/`--auth-pass`/`--auth-enabled`/`--tls-cert`/`--tls-key`/`--tls-enabled`/
`--session-secret` each get their own extra fallback tier too, consulted regardless of
`--driver` — these settings mean the same thing for every backend. Unlike the MongoDB tier
above, this one uses mongo-express's own real variable names verbatim
(`ME_CONFIG_BASICAUTH_USERNAME`, `ME_CONFIG_BASICAUTH_ENABLED`, `ME_CONFIG_SITE_SSL_CRT_PATH`,
`ME_CONFIG_SITE_SSL_ENABLED`, `ME_CONFIG_SITE_SESSIONSECRET`, ...), so a mongo-express
deployment's environment carries over unchanged, not just its `ME_`-prefixed variable naming
convention.

`--auth-enabled` and `--tls-enabled` default `true` and normally do nothing — gonosequel's own
convention is that basic auth/TLS activate from the mere presence of `--auth-user` or
`--tls-cert`+`--tls-key`, unlike mongo-express's own `ME_CONFIG_*_ENABLED` (which default
`false` and gate a username/password or cert/key that might otherwise sit unused). They exist so
an imported `ME_CONFIG_BASICAUTH_ENABLED=false`/`ME_CONFIG_SITE_SSL_ENABLED=false` is honored —
set explicitly to `false`, they clear the corresponding credentials/cert before gonosequel ever
looks at them, rather than being silently ignored.

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
| `--auth-user` / `--auth-pass` | `GNS_AUTH_USER` / `GNS_AUTH_PASS` (`ME_AUTH_USER` / `ME_AUTH_PASS`, then `ME_CONFIG_BASICAUTH_USERNAME` / `ME_CONFIG_BASICAUTH_PASSWORD`) | — | Basic auth in front of the web UI itself. |
| `--auth-enabled` | `GNS_AUTH_ENABLED` (`ME_AUTH_ENABLED`, then `ME_CONFIG_BASICAUTH_ENABLED`) | `true` | Whether `--auth-user`/`--auth-pass` take effect — see below. |
| `--tls-cert` / `--tls-key` | `GNS_TLS_CERT` / `GNS_TLS_KEY` (`ME_TLS_CERT` / `ME_TLS_KEY`, then `ME_CONFIG_SITE_SSL_CRT_PATH` / `ME_CONFIG_SITE_SSL_KEY_PATH`) | — | Serve HTTPS instead of plain HTTP, using this certificate and private key. Set both or neither — passing only one is a startup error. |
| `--tls-enabled` | `GNS_TLS_ENABLED` (`ME_TLS_ENABLED`, then `ME_CONFIG_SITE_SSL_ENABLED`) | `true` | Whether `--tls-cert`/`--tls-key` take effect — see below. |
| `--session-secret` | `GNS_SESSION_SECRET` (`ME_SESSION_SECRET`, then `ME_CONFIG_SITE_SESSIONSECRET`) | — | HMAC-signs the session ID handed out by `/api/connect` in `--sessions` mode — see [Server & sessions](/features/server-and-sessions). Unset means today's unsigned, opaque session IDs. |
| `--dev-proxy` | `GNS_DEV_PROXY` (`ME_DEV_PROXY`) | — | Reverse-proxy non-API requests to this URL instead of serving the embedded frontend. Used internally by `make dev`; prefer browsing Vite's own URL directly in development, since this path doesn't support the WebSocket upgrade hot module reload needs. |
| `--verbose` | `GNS_VERBOSE` (`ME_VERBOSE`) | `false` | Print extra diagnostic logging during startup and request handling (e.g. session connect/disconnect in `--sessions` mode). gonosequel-only, no mongo-express equivalent — no compat tier. |

Every flag also has a `--read-timeout` / `--write-timeout` pair (both default `30s`) for the
HTTP server's own timeouts, and standard Go `flag` package behavior applies: `-flag=value`,
`-flag value`, and `--flag` are all accepted, and `--help` prints this same list from the
binary itself.

## Startup banner

On every startup, gonosequel prints a banner with the effective configuration — driver,
connection target, bind address, and whether sessions mode/readonly/basic auth/TLS/the session
secret are on — regardless of `--verbose`. Any credential-shaped value (backend password, basic
auth password, session secret) is masked as `****`, never printed in the clear. `--verbose` adds
further lines beyond the banner (bookmarks directory, connect/disconnect lifecycle events), but
does not change what the banner itself shows.

## Example: read-only, bound to all interfaces, behind basic auth

```bash
./gonosequel \
  --url mongodb://localhost:27017 \
  --bind 0.0.0.0 --http-port 8081 \
  --readonly \
  --auth-user admin --auth-pass "$ADMIN_PASSWORD"
```

## Example: HTTPS with signed session IDs, in `--sessions` mode

```bash
./gonosequel \
  --sessions \
  --bind 0.0.0.0 --http-port 8443 \
  --tls-cert /etc/gonosequel/tls.crt --tls-key /etc/gonosequel/tls.key \
  --session-secret "$SESSION_SECRET"
```
