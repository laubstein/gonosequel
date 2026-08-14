// Package api wires the HTTP server: route registration, handlers, and
// middleware. It never talks to a concrete database driver directly — it
// depends only on the pkg/driver.Driver interface, resolved per request by
// pkg/session.Registry, so it has no idea whether the connection behind it
// is MongoDB or something else entirely.
package api

import (
	"errors"
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/basicauth"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/history"
	"github.com/laubstein/gonosequel/pkg/session"
)

// Version is the application version, set at build time via -ldflags.
var Version = "dev"

// Config holds the settings and shared state needed to build the Fiber
// app. Registry must be non-nil; in single-connection mode the caller
// connects at startup and registers it under session.DefaultID before
// calling New.
type Config struct {
	Registry *session.Registry
	// Driver is the backend the server was started against (e.g.
	// "mongodb"), reported by /api/info so the UI can show what it's
	// connected to.
	Driver   string
	Sessions bool // whether the UI may open additional connections
	Readonly bool
	AuthUser string
	AuthPass string

	// Assets, if non-nil, is the built frontend (web/dist) served for
	// every non-API route. Nil in tests that only exercise /api routes.
	Assets fs.FS
	// DevProxy, if set, reverse-proxies non-API routes to a Vite dev
	// server instead of serving Assets, enabling hot reload.
	DevProxy string

	// Docs, if non-nil, is the built documentation site
	// (docs/.vitepress/dist) served under /doc.
	Docs fs.FS

	// BookmarksDir is where saved connection bookmarks live (see
	// pkg/bookmarks). Empty disables the /api/bookmarks endpoint.
	BookmarksDir string
}

// deps bundles the dependencies handlers need, avoiding globals.
type deps struct {
	registry     *session.Registry
	history      *history.Store
	driver       string
	sessions     bool
	readonly     bool
	bookmarksDir string
}

// New builds a *fiber.App with middleware and routes registered, but does
// not start listening.
func New(cfg Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "gonosequel",
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(compress.New())

	if cfg.AuthUser != "" {
		app.Use(basicauth.New(basicauth.Config{
			Users: map[string]string{cfg.AuthUser: cfg.AuthPass},
		}))
	}

	if cfg.Readonly {
		app.Use(rejectWrites)
	}

	d := &deps{
		registry:     cfg.Registry,
		history:      history.NewStore(),
		driver:       cfg.Driver,
		sessions:     cfg.Sessions,
		readonly:     cfg.Readonly,
		bookmarksDir: cfg.BookmarksDir,
	}
	registerRoutes(app, d)
	// Registered before the SPA's own catch-all (inside registerAssets),
	// since Fiber matches "/*" patterns in registration order and the
	// broader SPA route would otherwise swallow every /doc/* request first.
	registerDocs(app, cfg.Docs)
	registerAssets(app, cfg.Assets, cfg.DevProxy)

	return app
}

// errorHandler converts any error returned by a handler into a uniform JSON
// body, so individual handlers never write error responses by hand. Known
// domain errors from pkg/client and pkg/session are mapped to the
// appropriate HTTP status.
func errorHandler(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
	}

	code := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, driver.ErrNotFound):
		code = fiber.StatusNotFound
	case errors.Is(err, driver.ErrAlreadyExists):
		code = fiber.StatusConflict
	case errors.Is(err, session.ErrNotFound):
		code = fiber.StatusBadRequest
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}

// rejectWrites enforces --readonly at the server boundary: any non-GET
// request to the API is refused with 403, regardless of what the UI shows.
func rejectWrites(c fiber.Ctx) error {
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
		return fiber.NewError(fiber.StatusForbidden, "server is running in readonly mode")
	}
	return c.Next()
}
