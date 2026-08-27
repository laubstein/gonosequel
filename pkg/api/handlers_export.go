package api

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/export"
)

// handleExport streams the full (unpaginated) result of a filter as JSON
// or CSV, writing directly from the query result to the response instead
// of building the whole export in memory first.
//
// This is the header-authenticated route, reachable from a script that can
// set X-Session-Id. A browser download cannot, so the UI goes through
// handleExportTicket/handleExportDownload instead.
func (d *deps) handleExport(c fiber.Ctx) error {
	format := c.Query("format", "json")
	if err := validateExportFormat(format); err != nil {
		return err
	}

	codec := currentClient(c)
	opts, err := parseFindOptions(c, codec)
	if err != nil {
		return err
	}

	return streamExport(c, codec, c.Params("db"), c.Params("coll"), format, opts)
}

// handleExportTicket issues a one-shot token the browser can navigate to,
// because a download navigation carries no X-Session-Id header and would
// otherwise resolve to the (nonexistent, in --sessions mode) default
// session. Registered as a GET on purpose: a POST would be rejected by
// rejectWrites and by the per-session readonly check, making export
// unusable in exactly the read-only mode it is most wanted in.
//
// The query is parsed here, not at download time, so an invalid filter
// comes back as an error the UI can show next to the button instead of
// replacing the page with a JSON error body.
func (d *deps) handleExportTicket(c fiber.Ctx) error {
	format := c.Query("format", "json")
	if err := validateExportFormat(format); err != nil {
		return err
	}
	if _, err := parseFindOptions(c, currentClient(c)); err != nil {
		return err
	}

	// Every string here must be cloned. Fiber's Params/Query return values
	// that alias the request's own buffer, which fasthttp reuses for the
	// next request — and a ticket by definition outlives the request that
	// issued it. Retaining them uncopied yields fragments of whatever
	// request comes next (in practice: pieces of the download URL's UUID).
	token, ok := d.exportTickets.issue(exportTicket{
		sessionID:  strings.Clone(currentSessionID(c)),
		db:         strings.Clone(c.Params("db")),
		coll:       strings.Clone(c.Params("coll")),
		format:     strings.Clone(format),
		filter:     strings.Clone(c.Query("filter")),
		projection: strings.Clone(c.Query("projection")),
		sort:       strings.Clone(c.Query("sort")),
		expires:    time.Now().Add(exportTicketTTL),
	})
	if !ok {
		return fiber.NewError(fiber.StatusTooManyRequests, "too many pending exports; try again in a moment")
	}
	return c.JSON(fiber.Map{"ticket": token})
}

// handleExportDownload redeems a ticket from handleExportTicket and
// streams the export.
//
// It is registered outside the withSession group because the whole point
// is that this request has no session header — the ticket names the
// session instead. That skips the per-session readonly check, which is
// fine: the check only rejects non-GET methods (see withSession), this is
// a GET, and the ticket grants nothing the session that issued it did not
// already have. Issuing went through withSession normally.
func (d *deps) handleExportDownload(c fiber.Ctx) error {
	t, ok := d.exportTickets.claim(c.Params("ticket"))
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "export link expired or already used")
	}

	codec, err := d.registry.Get(t.sessionID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "no active connection for this session; connect first")
	}

	opts, err := parseFindOptionsFrom(t.query, codec)
	if err != nil {
		return err
	}

	return streamExport(c, codec, t.db, t.coll, t.format, opts)
}

func validateExportFormat(format string) error {
	if format != "json" && format != "csv" {
		return fiber.NewError(fiber.StatusBadRequest, "format must be json or csv")
	}
	return nil
}

// streamExport is the shared body of both export routes, so the
// header-authenticated and ticket-authenticated paths cannot drift.
func streamExport(c fiber.Ctx, codec driver.Driver, db, coll, format string, opts driver.FindOptions) error {
	opts.Skip = 0
	opts.Limit = 0 // export ignores pagination: it exports every matching document

	result, err := codec.Find(c.Context(), db, coll, opts)
	if err != nil {
		return fmt.Errorf("export query: %w", err)
	}

	filename := fmt.Sprintf("%s.%s.%s", db, coll, format)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))

	if format == "csv" {
		c.Type("csv")
		return c.SendStreamWriter(func(w *bufio.Writer) {
			_ = export.CSV(w, result.Documents, codec)
		})
	}

	c.Type("json")
	return c.SendStreamWriter(func(w *bufio.Writer) {
		_ = export.JSON(w, result.Documents, codec)
	})
}
