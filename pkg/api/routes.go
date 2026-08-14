package api

import "github.com/gofiber/fiber/v3"

// registerRoutes mounts every route group on the app. Route handlers are
// implemented in the handlers_*.go files, grouped by domain.
func registerRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/info", handleInfo)
}
