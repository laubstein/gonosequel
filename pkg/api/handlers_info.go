package api

import "github.com/gofiber/fiber/v3"

// handleInfo reports the application version. It is unauthenticated and
// exists mainly as a liveness check.
func handleInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"app":     "mongo-express-go",
		"version": Version,
	})
}
