.PHONY: build build-web dev test test-short lint clean

GOFLAGS :=

build: web/dist
	go build -o mongo-express-go .

web/dist:
	cd web && npm ci && npm run build

dev:
	@echo "run 'cd web && npm run dev' in one shell, then:"
	go run . --dev-proxy http://localhost:5173

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
	rm -f mongo-express-go
	rm -rf web/dist
