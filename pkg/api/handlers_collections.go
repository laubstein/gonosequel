package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/mongo-express-go/pkg/client"
)

type createCollectionRequest struct {
	Name        string `json:"name"`
	Capped      bool   `json:"capped"`
	MaxSizeByte int64  `json:"maxSizeByte"`
	MaxDocs     int64  `json:"maxDocs"`
}

type renameCollectionRequest struct {
	NewName string `json:"newName"`
}

func (d *deps) handleListCollections(c fiber.Ctx) error {
	colls, err := currentClient(c).ListCollections(c.Context(), c.Params("db"))
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}
	return c.JSON(colls)
}

func (d *deps) handleCreateCollection(c fiber.Ctx) error {
	var req createCollectionRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	opts := client.CreateCollectionOptions{Capped: req.Capped, MaxSizeByte: req.MaxSizeByte, MaxDocs: req.MaxDocs}
	if err := currentClient(c).CreateCollection(c.Context(), c.Params("db"), req.Name, opts); err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true})
}

func (d *deps) handleDropCollection(c fiber.Ctx) error {
	if err := currentClient(c).DropCollection(c.Context(), c.Params("db"), c.Params("coll")); err != nil {
		return fmt.Errorf("drop collection: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (d *deps) handleRenameCollection(c fiber.Ctx) error {
	var req renameCollectionRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.NewName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "newName is required")
	}
	if err := currentClient(c).RenameCollection(c.Context(), c.Params("db"), c.Params("coll"), req.NewName); err != nil {
		return fmt.Errorf("rename collection: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (d *deps) handleCollectionStats(c fiber.Ctx) error {
	stats, err := currentClient(c).Stats(c.Context(), c.Params("db"), c.Params("coll"))
	if err != nil {
		return fmt.Errorf("collection stats: %w", err)
	}
	return c.JSON(stats)
}

func (d *deps) handleCollectionSchema(c fiber.Ctx) error {
	fields, err := currentClient(c).InferSchema(c.Context(), c.Params("db"), c.Params("coll"), 0)
	if err != nil {
		return fmt.Errorf("infer schema: %w", err)
	}
	return c.JSON(fields)
}
