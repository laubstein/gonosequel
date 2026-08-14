package api

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/mongo-express-go/pkg/export"
)

// handleExport streams the full (unpaginated) result of a filter as JSON
// or CSV, writing directly from the query result to the response instead
// of building the whole export in memory first.
func (d *deps) handleExport(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	format := c.Query("format", "json")
	if format != "json" && format != "csv" {
		return fiber.NewError(fiber.StatusBadRequest, "format must be json or csv")
	}

	opts, err := parseFindOptions(c)
	if err != nil {
		return err
	}
	opts.Skip = 0
	opts.Limit = 0 // export ignores pagination: it exports every matching document

	result, err := currentClient(c).Find(c.Context(), db, coll, opts)
	if err != nil {
		return fmt.Errorf("export query: %w", err)
	}

	filename := fmt.Sprintf("%s.%s.%s", db, coll, format)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))

	if format == "csv" {
		c.Type("csv")
		return c.SendStreamWriter(func(w *bufio.Writer) {
			_ = export.CSV(w, result.Documents)
		})
	}

	c.Type("json")
	return c.SendStreamWriter(func(w *bufio.Writer) {
		_ = export.JSON(w, result.Documents)
	})
}
