package api

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// registerAssets wires either a served frontend build (production, dist
// non-nil), a reverse proxy to the Vite dev server (devProxy set), or
// nothing at all (dist nil and devProxy empty — the case in tests, which
// only exercise /api routes). It must be registered after every /api
// route, since it matches everything under "/*".
//
// devProxy is not the recommended dev workflow (`make dev` doesn't use
// it): this is a plain HTTP passthrough, so it doesn't forward the
// WebSocket upgrade Vite's hot module reload needs — a full navigate
// still gets current code, but state isn't preserved across edits the
// way it is when browsing Vite's own URL directly, which proxies /api to
// this server instead (see web/vite.config.ts) and has real WebSocket
// support. Kept for cases where single-origin access matters more than
// HMR fidelity.
func registerAssets(app *fiber.App, dist fs.FS, devProxy string) {
	switch {
	case devProxy != "":
		// --dev-proxy only ever targets localhost:<vite-port> on the
		// developer's own machine, never a production deployment, so the
		// SSRF guard against loopback/private addresses that Fiber's proxy
		// applies by default would otherwise block every request here.
		proxy.WithSecurityPolicy(proxy.SecurityPolicy{
			AllowedSchemes:  []string{"http", "https"},
			AllowPrivateIPs: true,
		})
		// proxy.Forward sends every request to devProxy verbatim,
		// dropping the original path — fine for a single fixed upstream
		// page, wrong here since Vite needs the real request path (e.g.
		// /@vite/client, /src/main.tsx) to serve the right module.
		// proxy.Do takes the full target per call, so the path is
		// rebuilt from the incoming request each time.
		app.Get("/*", func(c fiber.Ctx) error {
			return proxy.Do(c, devProxy+c.OriginalURL())
		})
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
