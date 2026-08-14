package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// handleCollectionsOverview reports size/count stats for every collection
// in a database, for spotting bloated or unusually large collections.
func (d *deps) handleCollectionsOverview(c fiber.Ctx) error {
	overview, err := currentClient(c).CollectionsOverview(c.Context(), c.Params("db"))
	if err != nil {
		return fmt.Errorf("collections overview: %w", err)
	}
	return c.JSON(overview)
}

// handleIndexUsage reports $indexStats for every index in every collection
// of a database, for spotting indexes that have never been used.
func (d *deps) handleIndexUsage(c fiber.Ctx) error {
	usage, err := currentClient(c).IndexUsage(c.Context(), c.Params("db"))
	if err != nil {
		return fmt.Errorf("index usage: %w", err)
	}
	return c.JSON(usage)
}

// handleCurrentOps reports active server operations running for at least
// ?minSecs= (default 1) seconds. This is instance-wide, not scoped to a
// single database — each operation's namespace shows which database and
// collection it belongs to.
func (d *deps) handleCurrentOps(c fiber.Ctx) error {
	minSecs := queryInt64(c, "minSecs", 1)
	ops, err := currentClient(c).CurrentOps(c.Context(), minSecs)
	if err != nil {
		return fmt.Errorf("current ops: %w", err)
	}
	return c.JSON(ops)
}
