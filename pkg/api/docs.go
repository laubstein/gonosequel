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
	app.Get("/doc/*", static.New("", static.Config{
		FS:              docs,
		NotFoundHandler: serveIndex(docs),
	}))
}
