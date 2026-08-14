package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type createDatabaseRequest struct {
	Name              string `json:"name"`
	InitialCollection string `json:"initialCollection"`
}

func (d *deps) handleListDatabases(c fiber.Ctx) error {
	dbs, err := currentClient(c).ListDatabases(c.Context())
	if err != nil {
		return fmt.Errorf("list databases: %w", err)
	}
	return c.JSON(dbs)
}

func (d *deps) handleCreateDatabase(c fiber.Ctx) error {
	var req createDatabaseRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if err := currentClient(c).CreateDatabase(c.Context(), req.Name, req.InitialCollection); err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true})
}

func (d *deps) handleDropDatabase(c fiber.Ctx) error {
	if err := currentClient(c).DropDatabase(c.Context(), c.Params("db")); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
