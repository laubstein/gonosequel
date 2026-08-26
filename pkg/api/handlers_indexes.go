package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// indexKeyEntry is one field of an index's key spec. An ordered slice, not
// a map[string]field->direction — Go's JSON decoder into a map loses field
// order, and for a compound index the key order is semantically
// significant (it decides which sorts/range queries the index can serve).
type indexKeyEntry struct {
	Field     string `json:"field"`
	Direction int    `json:"direction"`
}

type createIndexRequest struct {
	Keys                    []indexKeyEntry `json:"keys"`
	Unique                  bool            `json:"unique"`
	Sparse                  bool            `json:"sparse"`
	ExpireAfterSeconds      *int64          `json:"expireAfterSeconds"`
	PartialFilterExpression map[string]any  `json:"partialFilterExpression"`
}

type updateIndexTTLRequest struct {
	ExpireAfterSeconds int64 `json:"expireAfterSeconds"`
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
	for _, k := range req.Keys {
		keys = append(keys, driver.Entry{Key: k.Field, Value: k.Direction})
	}

	opts := driver.CreateIndexOptions{
		Unique:             req.Unique,
		Sparse:             req.Sparse,
		ExpireAfterSeconds: req.ExpireAfterSeconds,
	}
	if req.PartialFilterExpression != nil {
		opts.PartialFilterExpression = driver.Doc(req.PartialFilterExpression)
	}

	name, err := currentClient(c).CreateIndex(c.Context(), c.Params("db"), c.Params("coll"), keys, opts)
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

func (d *deps) handleUpdateIndexTTL(c fiber.Ctx) error {
	var req updateIndexTTLRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	err := currentClient(c).UpdateIndexTTL(c.Context(), c.Params("db"), c.Params("coll"), c.Params("name"), req.ExpireAfterSeconds)
	if err != nil {
		return fmt.Errorf("update index ttl: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
