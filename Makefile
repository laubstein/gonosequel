.PHONY: build docs-dev dev dev-down test test-short lint clean

GOFLAGS :=
MONGO_PORT ?= 27017
HTTP_PORT ?= 8081
VITE_PORT ?= 5173

# gofmt lives in $GOROOT/bin, which isn't always on PATH even when `go`
# itself is (e.g. a bare toolchain install without its bin dir exported).
export PATH := $(PATH):$(shell go env GOROOT)/bin

build: web/dist docs/.vitepress/dist
	CGO_ENABLED=0 go build -o gonosequel .

web/dist:
	cd web && npm ci && npm run build

docs/.vitepress/dist:
	cd docs && npm ci && npm run docs:build

# Live-editing the documentation itself — not part of `make dev`, which is
# about developing the app, not the docs. Runs on its own, unrelated to
# scripts/dev.sh.
docs-dev:
	cd docs && npm ci && npm run docs:dev

# Brings up a full local dev environment: a MongoDB (reusing one already
# listening on MONGO_PORT, otherwise starting one in Docker), the Go API
# server, and the Vite dev server — open the URL it prints, not the API
# server's port. See scripts/dev.sh for the details.
dev:
	@MONGO_PORT=$(MONGO_PORT) HTTP_PORT=$(HTTP_PORT) VITE_PORT=$(VITE_PORT) ./scripts/dev.sh

# Removes the MongoDB container make dev may have started, if any. `docker
# rm -f` on a name that doesn't exist still exits 0 on this Docker version,
# so there's no reliable way to report whether anything was actually there.
dev-down:
	@docker rm -f gonosequel-dev-mongo >/dev/null 2>&1; echo "dev MongoDB container removed if it existed"

test:
	go test $(GOFLAGS) -race ./...

test-short:
	go test $(GOFLAGS) -short ./...

lint:
	test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...
	command -v staticcheck >/dev/null && staticcheck ./... || echo "staticcheck not installed, skipping"
	command -v errcheck >/dev/null && errcheck ./... || echo "errcheck not installed, skipping"

clean:
	rm -f gonosequel
	rm -rf web/dist
	rm -rf docs/.vitepress/dist docs/.vitepress/cache
