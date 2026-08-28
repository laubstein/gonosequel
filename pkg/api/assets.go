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
		// "/" and "/index.html" are registered explicitly, ahead of the
		// wildcard static handler below, so they never go through
		// static.New's own file-serving path for those two exact
		// paths — see serveIndex for why that matters.
		index := serveIndex(dist)
		app.Get("/", index)
		app.Get("/index.html", index)
		app.Get("/*", static.New("", static.Config{
			FS:              dist,
			NotFoundHandler: index,
		}))
	}
}

// serveIndex returns index.html, for a path the static handler didn't
// match (client-side routing surviving a hard reload on an inner route)
// and, deliberately, for "/" and "/index.html" themselves rather than
// letting static.New serve those two paths as ordinary files.
//
// The embed package reports every embedded file's ModTime as the zero
// time, identically across every build — verified against the stdlib.
// fasthttp's static file server uses that ModTime for If-Modified-Since
// handling, so after the binary is rebuilt and restarted, a browser tab's
// next reload
// sends back the same (zero) Last-Modified it saw before and gets a bare
// 304, telling it to keep using its own cached copy of index.html. That
// cached HTML references the previous build's content-hashed JS/CSS
// filenames, which no longer exist — so the page never loads the new
// bundle, and every subsequent reload repeats the same 304, forever. The
// hashed asset files themselves don't have this problem (a change in
// content always means a new filename, so a stale cache entry is simply
// never looked up again) — only the unhashed entry document does, hence
// fixing it here specifically rather than disabling caching everywhere.
func serveIndex(dist fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		data, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "index.html not found in embedded assets")
		}
		// The static handler already set a 404 status on the underlying
		// response before calling us; reset it to 200 since this is a
		// legitimate SPA route, not a missing asset.
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.Status(fiber.StatusOK).Type("html").Send(data)
	}
}
