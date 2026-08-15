# Getting started

Go NoSequel connects to **MongoDB** or **Redis/Valkey** — pick with `--driver` (see
[CLI reference](/cli-reference)) or the connection screen's own database-type selector in
`--sessions` mode. See [Supported databases](/#supported-databases) for what's next.

## Requirements

- Go 1.26+
- Node.js 24+ (only to build the frontend and this documentation site — not needed to run the
  compiled binary)
- Docker (only for `make test`'s integration tests, and optionally for `make dev` if you don't
  already have a MongoDB running locally)

## Development

```bash
make dev
```

This single command:

1. Looks for a MongoDB already listening on `localhost:27017`. If it doesn't find one and
   Docker is installed, it starts one in a container, reused across runs so your data
   persists between `make dev` sessions until you run `make dev-down`.
2. Builds and starts the Go API server.
3. Starts the Vite dev server and prints the URL to open — use that URL, not the API server's
   port. Vite proxies `/api` requests to the Go server and supports the WebSocket upgrade hot
   module reload needs, which the Go server's own `--dev-proxy` fallback does not.

Press <kbd>Ctrl+C</kbd> to stop everything. Override the ports if the defaults collide with
something else on your machine:

```bash
make dev MONGO_PORT=27018 HTTP_PORT=8082 VITE_PORT=5174
```

`make dev-down` removes the MongoDB container `make dev` may have started, if you want to
clear out its data.

`make dev` is MongoDB-specific. To develop against Redis/Valkey, run a server yourself and
point the built binary at it directly: `./gonosequel --driver redis --url redis://localhost:6379`.

## Building for production

```bash
make build
```

Builds the frontend (`web/dist`) and this documentation site (`docs/.vitepress/dist`), then
embeds both into a single `gonosequel` binary via `go:embed`. Run it against MongoDB (the
default) or Redis/Valkey:

```bash
./gonosequel --url mongodb://user:pass@host:27017
./gonosequel --driver redis --url redis://user:pass@host:6379
```

The app is served at `/`, this documentation at `/doc`.

## Testing

```bash
make test        # unit + integration; starts and stops MongoDB and Redis containers itself via Docker
make test-short   # unit tests only, no Docker required
make lint         # gofmt, go vet, staticcheck, errcheck
```

See [Contributing](/contributing) for the package layout and coding conventions if you're
changing the Go or React source.
