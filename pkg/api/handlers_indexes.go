package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/driver"
)

type createIndexRequest struct {
	// Keys maps field name to direction (1 or -1), matching MongoDB's own
	// index key document shape.
	Keys   map[string]int `json:"keys"`
	Unique bool           `json:"unique"`
}

func (d *deps) handleListIndexes(c fiber.Ctx) error {
	indexes, err := currentClient(c).ListIndexes(c.Context(), c.Params("db"), c.Params("coll"))
	if err != nil {
		return fmt.Errorf("list indexes: %w", err)
	}
	return c.JSON(indexes)
}

func (d *deps) handleCreateIndex(c fiber.Ctx) error {
	var req createIndexRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if len(req.Keys) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "keys is required")
	}

	keys := driver.OrderedDoc{}
	for field, dir := range req.Keys {
		keys = append(keys, driver.Entry{Key: field, Value: dir})
	}

	name, err := currentClient(c).CreateIndex(c.Context(), c.Params("db"), c.Params("coll"), keys, req.Unique)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"name": name})
}

func (d *deps) handleDropIndex(c fiber.Ctx) error {
	err := currentClient(c).DropIndex(c.Context(), c.Params("db"), c.Params("coll"), c.Params("name"))
	if err != nil {
		return fmt.Errorf("drop index: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
