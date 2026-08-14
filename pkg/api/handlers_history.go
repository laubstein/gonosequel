package api

import "github.com/gofiber/fiber/v3"

func (d *deps) handleHistory(c fiber.Ctx) error {
	return c.JSON(d.history.List(currentSessionID(c)))
}
