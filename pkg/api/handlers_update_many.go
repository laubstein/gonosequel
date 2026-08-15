package api

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type updateManyRequest struct {
	Filter json.RawMessage `json:"filter"`
	Update json.RawMessage `json:"update"`
}

// handleUpdateMany applies an update document to every document matching
// a filter — MongoDB's updateMany. Like aggregate and command, this is a
// POST so --readonly's middleware rejects it outright.
func (d *deps) handleUpdateMany(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)

	var req updateManyRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	filter, err := codec.UnmarshalDoc(req.Filter)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid filter: "+err.Error())
	}
	update, err := codec.UnmarshalDoc(req.Update)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid update: "+err.Error())
	}

	matched, modified, err := codec.UpdateMany(c.Context(), db, coll, filter, update)
	if err != nil {
		return fmt.Errorf("update many: %w", err)
	}
	return c.JSON(fiber.Map{"matched": matched, "modified": modified})
}
