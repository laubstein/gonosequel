package api

import "github.com/gofiber/fiber/v3"

// handleInfo reports the application version and whether the server is
// running in --readonly mode, so the frontend can show a banner without
// having to first attempt a write and observe it get rejected. It is
// unauthenticated and also serves as a liveness check.
func (d *deps) handleInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"app":      "gonosequel",
		"version":  Version,
		"readonly": d.readonly,
	})
}
