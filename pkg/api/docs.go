package api

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// registerDocs serves the built VitePress documentation site
// (docs/.vitepress/dist) under /doc. A no-op when docs is nil — the case
// in tests, and for anyone building without running `make build` (docs
// aren't required for the app itself to work).
func registerDocs(app *fiber.App, docs fs.FS) {
	if docs == nil {
		return
	}
	// "/doc" and "/doc/index.html" bypass static.New's own file serving
	// for the same reason as the app shell in assets.go: go:embed's
	// identical zero ModTime across every build makes fasthttp's
	// If-Modified-Since check return a false 304 after a rebuild, pinning
	// a browser tab to the previous docs build forever.
	index := serveIndex(docs)
	app.Get("/doc", index)
	app.Get("/doc/index.html", index)
	app.Get("/doc/*", static.New("", static.Config{
		FS:              docs,
		NotFoundHandler: index,
	}))
}
