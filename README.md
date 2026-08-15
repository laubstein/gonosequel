# Go NoSequel

<img src="gopher.png" alt="Go NoSequel gopher mascot" width="200" align="right">

A web-based MongoDB explorer, in the spirit of [pgweb](https://github.com/sosedoff/pgweb)'s
interface and [mongo-express](https://github.com/mongo-express/mongo-express)'s feature set.
Ships as a single self-contained binary — the React frontend is built and embedded into the
Go executable, so there's nothing else to install or run alongside it in production.

Browse databases and collections, view/create/edit/delete documents as Extended JSON, manage
indexes, run filter queries with schema-aware autocomplete and `explain()`, export results as
JSON/CSV, and connect to multiple MongoDB instances at once via saved bookmarks. The UI is in
English by default and switches to Portuguese automatically when the browser asks for it.

## Credits

Go NoSequel exists thanks to two projects it borrows heavily from: [pgweb](https://github.com/sosedoff/pgweb)
for the interface and overall usability, and [mongo-express](https://github.com/mongo-express/mongo-express)
for the feature set it aims to cover. Both are MIT-licensed, and this project is grateful for
the ideas and is released under the same terms — see [License](#license).

## Requirements

- Go 1.26+
- Node.js 24+ (only to build the frontend; not needed to run the compiled binary)
- Docker (only for running the test suite's integration tests, and optionally for `make dev`
  if you don't already have a MongoDB running locally)

## Quick start (development)

```bash
make dev
```

This single command:

1. Looks for a MongoDB already listening on `localhost:27017`. If it doesn't find one and
   Docker is installed, it starts one in a container (reused across runs, so your data
   persists between `make dev` sessions until you run `make dev-down`).
2. Builds and starts the Go API server.
3. Starts the Vite dev server and prints the URL to open — **use that URL, not the API
   server's port**: Vite proxies `/api` requests to the Go server itself and supports hot
   module reload, so editing frontend source updates the page instantly.

Press Ctrl+C to stop everything. `make dev-down` additionally removes the MongoDB container
`make dev` may have started, if you want to clear out its data.

Override the ports if the defaults collide with something else on your machine:

```bash
make dev MONGO_PORT=27018 HTTP_PORT=8082 VITE_PORT=5174
```

## Building for production

```bash
make build
```

Builds the frontend (`web/dist`) and embeds it into a single `gonosequel` binary via
`go:embed`. Run it against any MongoDB:

```bash
./gonosequel --url mongodb://user:pass@host:27017
```

## CLI flags

Every env var below is read as `GNS_<name>` first; if unset, `ME_<name>` (mongo-express's own
naming — this project's earlier working name) is used as a fallback, so an existing
mongo-express deployment can switch images without also rewriting its environment. A value
containing `$VAR` or `${VAR}` is expanded against the rest of the environment before use (the
same substitution a shell does), so e.g. `GNS_URL=$MONGO_URL` resolves through to whatever
`MONGO_URL` is set to — handy when a value is already wired up by the surrounding deployment
(a docker-compose service link, a Kubernetes downstream env var, etc).

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--url` | `GNS_URL` (`ME_URL`) | — | Full MongoDB connection URL. Takes priority over `--host`/`--port`/`--user`/`--pass`/`--db`. |
| `--host` | `GNS_HOST` (`ME_HOST`) | `localhost` | MongoDB host, used when `--url` isn't given. |
| `--port` | `GNS_MONGO_PORT` (`ME_MONGO_PORT`) | `27017` | MongoDB port. |
| `--user` / `--pass` | `GNS_USER` / `GNS_PASS` (`ME_USER` / `ME_PASS`) | — | MongoDB credentials. |
| `--db` | `GNS_DB` (`ME_DB`) | — | Default database. |
| `--bookmark` | `GNS_BOOKMARK` (`ME_BOOKMARK`) | — | Load the connection from a saved bookmark instead of the flags above (see below). Takes priority over `--url`. |
| `--bind` | `GNS_BIND` (`ME_BIND`) | `127.0.0.1` | Address the HTTP server binds to. |
| `--http-port` | `GNS_HTTP_PORT` (`ME_HTTP_PORT`) | `8081` | HTTP server port. |
| `--sessions` | `GNS_SESSIONS` (`ME_SESSIONS`) | `false` | Multi-session mode: don't connect at startup: the UI prompts to connect (by URL or bookmark), and can hold several connections open at once. |
| `--readonly` | `GNS_READONLY` (`ME_READONLY`) | `false` | Reject every non-GET API request with 403, enforced server-side. |
| `--auth-user` / `--auth-pass` | `GNS_AUTH_USER` / `GNS_AUTH_PASS` (`ME_AUTH_USER` / `ME_AUTH_PASS`) | — | Basic auth in front of the web UI itself. |
| `--dev-proxy` | `GNS_DEV_PROXY` (`ME_DEV_PROXY`) | — | Reverse-proxy non-API requests to this URL instead of serving the embedded frontend. Used internally by `make dev`; prefer browsing Vite's own URL directly, since this path doesn't support the WebSocket upgrade hot module reload needs. |

## Bookmarks

Save a named connection so you don't have to retype it:

```toml
# ~/.gonosequel/bookmarks/prod.toml
url = "mongodb://user:pass@prod.example.com:27017"
```

```bash
./gonosequel --bookmark prod
```

In `--sessions` mode, saved bookmarks also show up as one-click options in the connection
screen — the URL (and any password in it) is resolved server-side and never sent to the
browser.

## Testing

```bash
make test          # unit + integration; starts and stops a MongoDB container itself via Docker
make test-short     # unit tests only, no Docker required
make lint           # gofmt, go vet, staticcheck, errcheck
```

## Project layout

See [`AGENTS.md`](AGENTS.md) for the package layout, coding conventions, and the invariants
around Extended JSON handling that matter most if you're changing `pkg/client` or `pkg/api`.
[`PLAN.md`](PLAN.md) has the original design rationale and phased build order.

## License

MIT — see [`LICENSE`](LICENSE). Same license as [pgweb](https://github.com/sosedoff/pgweb)
and [mongo-express](https://github.com/mongo-express/mongo-express), the two projects this
one is built on top of.
