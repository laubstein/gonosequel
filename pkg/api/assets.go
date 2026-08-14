package api

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// registerAssets wires either a served frontend build (production, dist
// non-nil), a reverse proxy to the Vite dev server (devProxy set, for hot
// reload without rebuilding the Go binary), or nothing at all (dist nil
// and devProxy empty — the case in tests, which only exercise /api
// routes). It must be registered after every /api route, since it matches
// everything under "/*".
func registerAssets(app *fiber.App, dist fs.FS, devProxy string) {
	switch {
	case devProxy != "":
		app.Get("/*", proxy.Forward(devProxy))
	case dist != nil:
		app.Get("/*", static.New("", static.Config{
			FS:              dist,
			NotFoundHandler: serveIndex(dist),
		}))
	}
}

// serveIndex returns index.html for any path the static handler didn't
// match, so client-side routing survives a hard reload on an inner route.
func serveIndex(dist fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		data, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "index.html not found in embedded assets")
		}
		// The static handler already set a 404 status on the underlying
		// response before calling us; reset it to 200 since this is a
		// legitimate SPA route, not a missing asset.
		return c.Status(fiber.StatusOK).Type("html").Send(data)
	}
}
