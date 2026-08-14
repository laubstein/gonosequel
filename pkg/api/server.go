// Package api wires the HTTP server: route registration, handlers, and
// middleware. It never talks to the MongoDB driver directly — that lives in
// pkg/client — and it declares only the narrow interfaces it needs from its
// dependencies.
package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// Version is the application version, set at build time via -ldflags.
var Version = "dev"

// Config holds the settings needed to build the Fiber app.
type Config struct {
	Readonly bool
	AuthUser string
	AuthPass string
}

// New builds a *fiber.App with middleware and routes registered, but does
// not start listening.
func New(cfg Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "mongo-express-go",
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(compress.New())

	if cfg.Readonly {
		app.Use(rejectWrites)
	}

	registerRoutes(app)

	return app
}

// errorHandler converts any error returned by a handler into a uniform JSON
// body, so individual handlers never write error responses by hand.
func errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fe, ok := err.(*fiber.Error); ok {
		code = fe.Code
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
