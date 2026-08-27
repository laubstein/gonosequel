package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/history"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 1000
)

// handleListDocuments runs a find query with filter/sort/projection and
// pagination, returning documents as relaxed Extended JSON. It writes the
// serialized body directly with Send, never c.JSON, since a second JSON
// marshaling pass over an already-serialized document would destroy the
// BSON type information Extended JSON exists to preserve.
func (d *deps) handleListDocuments(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")

	codec := currentClient(c)
	opts, err := parseFindOptions(c, codec)
	if err != nil {
		return err
	}

	result, err := codec.Find(c.Context(), db, coll, opts)
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}

	// Only log queries with an actual filter; recording every unfiltered
	// page load would drown out queries worth replaying.
	if rawFilter := c.Query("filter"); rawFilter != "" && rawFilter != "{}" {
		d.history.Add(currentSessionID(c), history.Entry{
			Database:   db,
			Collection: coll,
			Filter:     rawFilter,
			At:         time.Now().Format(time.RFC3339),
		})
	}

	body, err := marshalListResponse(codec, result)
	if err != nil {
		return fmt.Errorf("marshal documents: %w", err)
	}
	return c.Type("json").Send(body)
}

func parseFindOptions(c fiber.Ctx, codec driver.DocCodec) (driver.FindOptions, error) {
	return parseFindOptionsFrom(c.Query, codec)
}

// parseFindOptionsFrom is parseFindOptions over any query lookup with
// fiber.Ctx.Query's signature. The export download path reads its query
// text out of a one-shot ticket rather than the URL (see
// export_tickets.go), and must interpret it exactly the same way.
func parseFindOptionsFrom(query func(string, ...string) string, codec driver.DocCodec) (driver.FindOptions, error) {
	var opts driver.FindOptions

	if raw := query("filter"); raw != "" {
		filter, err := codec.UnmarshalDoc([]byte(raw))
		if err != nil {
			return opts, fiber.NewError(fiber.StatusBadRequest, "invalid filter: "+err.Error())
		}
		opts.Filter = filter
	}
	if raw := query("projection"); raw != "" {
		proj, err := codec.UnmarshalDoc([]byte(raw))
		if err != nil {
			return opts, fiber.NewError(fiber.StatusBadRequest, "invalid projection: "+err.Error())
		}
		opts.Projection = proj
	}
	if raw := query("sort"); raw != "" {
		sortDoc, err := codec.UnmarshalDoc([]byte(raw))
		if err != nil {
			return opts, fiber.NewError(fiber.StatusBadRequest, "invalid sort: "+err.Error())
		}
		for k, v := range sortDoc {
			opts.Sort = append(opts.Sort, driver.Entry{Key: k, Value: v})
		}
	}

	opts.Skip = parseInt64(query("skip"), 0)
	opts.Limit = parseInt64(query("limit"), defaultPageLimit)
	if opts.Limit > maxPageLimit {
		opts.Limit = maxPageLimit
	}

	return opts, nil
}

func queryInt64(c fiber.Ctx, key string, def int64) int64 {
	return parseInt64(c.Query(key), def)
}

// parseInt64 returns raw as an int64, falling back to def when it is empty
// or unparseable — a malformed skip/limit is treated as absent rather than
// failing the request.
func parseInt64(raw string, def int64) int64 {
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func marshalListResponse(codec driver.DocCodec, result driver.FindResult) ([]byte, error) {
	docs := make([]driver.Doc, len(result.Documents))
	copy(docs, result.Documents)
	wrapper := driver.Doc{
		"documents":       docs,
		"total":           result.Total,
		"totalIsEstimate": result.TotalIsEstimate,
	}
	return codec.MarshalRelaxed(wrapper)
}

func (d *deps) handleGetDocument(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)
	id, err := codec.DecodeDocID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid document id")
	}

	doc, err := codec.FindOne(c.Context(), db, coll, id)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}

	// Editing form: canonical, so a save round-trip cannot change a
	// value's BSON type (e.g. Long silently becoming Double).
	body, err := codec.MarshalCanonical(doc)
	if err != nil {
		return fmt.Errorf("marshal document: %w", err)
	}
	return c.Type("json").Send(body)
}

func (d *deps) handleInsertDocument(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)

	doc, err := codec.UnmarshalDoc(c.Body())
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid document: "+err.Error())
	}

	id, err := codec.InsertOne(c.Context(), db, coll, doc)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	encodedID, err := codec.EncodeDocID(id)
	if err != nil {
		return fmt.Errorf("encode inserted id: %w", err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": encodedID})
}

func (d *deps) handleReplaceDocument(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)
	id, err := codec.DecodeDocID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid document id")
	}

	doc, err := codec.UnmarshalDoc(c.Body())
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid document: "+err.Error())
	}

	if err := codec.ReplaceOne(c.Context(), db, coll, id, doc); err != nil {
		return fmt.Errorf("replace document: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (d *deps) handleDeleteDocument(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)
	id, err := codec.DecodeDocID(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid document id")
	}

	if err := codec.DeleteOne(c.Context(), db, coll, id); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// handleExplainQuery runs the current filter through the query planner and
// returns the raw explain document as relaxed Extended JSON.
func (d *deps) handleExplainQuery(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")
	codec := currentClient(c)

	var filter driver.Doc
	if raw := c.Query("filter"); raw != "" {
		f, err := codec.UnmarshalDoc([]byte(raw))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid filter: "+err.Error())
		}
		filter = f
	}

	result, err := codec.Explain(c.Context(), db, coll, filter)
	if err != nil {
		return fmt.Errorf("explain: %w", err)
	}

	body, err := codec.MarshalRelaxed(result)
	if err != nil {
		return fmt.Errorf("marshal explain result: %w", err)
	}
	return c.Type("json").Send(body)
}
