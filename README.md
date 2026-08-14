# Go NoSequel

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

| Flag | Env var | Default | Description |
|---|---|---|---|
| `--url` | `ME_URL` | — | Full MongoDB connection URL. Takes priority over `--host`/`--port`/`--user`/`--pass`/`--db`. |
| `--host` | `ME_HOST` | `localhost` | MongoDB host, used when `--url` isn't given. |
| `--port` | `ME_MONGO_PORT` | `27017` | MongoDB port. |
| `--user` / `--pass` | `ME_USER` / `ME_PASS` | — | MongoDB credentials. |
| `--db` | `ME_DB` | — | Default database. |
| `--bookmark` | `ME_BOOKMARK` | — | Load the connection from a saved bookmark instead of the flags above (see below). Takes priority over `--url`. |
| `--bind` | `ME_BIND` | `127.0.0.1` | Address the HTTP server binds to. |
| `--http-port` | `ME_HTTP_PORT` | `8081` | HTTP server port. |
| `--sessions` | `ME_SESSIONS` | `false` | Multi-session mode: don't connect at startup: the UI prompts to connect (by URL or bookmark), and can hold several connections open at once. |
| `--readonly` | `ME_READONLY` | `false` | Reject every non-GET API request with 403, enforced server-side. |
| `--auth-user` / `--auth-pass` | `ME_AUTH_USER` / `ME_AUTH_PASS` | — | Basic auth in front of the web UI itself. |
| `--dev-proxy` | `ME_DEV_PROXY` | — | Reverse-proxy non-API requests to this URL instead of serving the embedded frontend. Used internally by `make dev`; prefer browsing Vite's own URL directly, since this path doesn't support the WebSocket upgrade hot module reload needs. |

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
