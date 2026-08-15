package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/laubstein/gonosequel/pkg/bookmarks"
	"github.com/laubstein/gonosequel/pkg/session"
)

type connectRequest struct {
	// URL is a full connection string (mongodb://... or redis://...),
	// used as-is.
	URL string `json:"url"`
	// Driver names which backend URL connects to (e.g. "mongodb",
	// "redis"). Optional: if empty, inferred from URL's scheme, so
	// bookmarks (which don't store a driver) and the raw-URL entry mode
	// still work.
	Driver string `json:"driver"`
	// Bookmark, if set instead of URL, is resolved server-side from
	// bookmarksDir — the saved URL (password included) never round-trips
	// through the browser, unlike URL above which the client typed in.
	Bookmark string `json:"bookmark"`
	Name     string `json:"name"`
}

// inferDriver guesses the backend from a connection URL's scheme, for
// callers (bookmarks, the raw-URL entry mode) that don't send an explicit
// Driver. Redis and Valkey share the redis:// scheme and are
// wire-compatible, so either name would dial identically — "redis" is
// just the default label.
func inferDriver(rawURL string) string {
	if strings.HasPrefix(rawURL, "redis://") || strings.HasPrefix(rawURL, "rediss://") {
		return "redis"
	}
	return "mongodb"
}

// handleConnect opens a new MongoDB connection and registers it as a
// session. In single-connection mode (Sessions=false) this replaces the
// default session; in --sessions mode it adds a new one, letting the UI
// hold several connections open at once.
func (d *deps) handleConnect(c fiber.Ctx) error {
	var req connectRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	targetURL := req.URL
	driverName := req.Driver
	displayName := req.Name
	if req.Bookmark != "" {
		b, err := bookmarks.Load(d.bookmarksDir, req.Bookmark)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "unknown bookmark: "+req.Bookmark)
		}
		targetURL = b.URL
		driverName = inferDriver(b.URL)
		if displayName == "" {
			displayName = b.Name
		}
	}
	if targetURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url or bookmark is required")
	}
	if driverName == "" {
		driverName = inferDriver(targetURL)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	cl, err := d.connect(ctx, driverName, targetURL)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("connect: %v", err))
	}

	id := session.DefaultID
	if d.sessions {
		id = uuid.NewString()
	}

	if displayName == "" {
		displayName = redactURI(targetURL)
	}

	d.registry.Put(id, cl, session.Info{ID: id, URI: redactURI(targetURL), Name: displayName, Driver: driverName})

	return c.JSON(fiber.Map{"sessionId": id})
}

// handleDisconnect closes and removes a session's connection.
func (d *deps) handleDisconnect(c fiber.Ctx) error {
	id := c.Get(sessionIDHeader)
	if id == "" {
		id = session.DefaultID
	}
	if err := d.registry.Remove(c.Context(), id); err != nil {
		return fmt.Errorf("disconnect: %w", err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// handleListSessions lists every active connection (single entry outside
// --sessions mode).
func (d *deps) handleListSessions(c fiber.Ctx) error {
	return c.JSON(d.registry.List())
}

// handleConnectionInfo reports the current session's connection details.
func (d *deps) handleConnectionInfo(c fiber.Ctx) error {
	id := currentSessionID(c)
	for _, info := range d.registry.List() {
		if info.ID == id {
			return c.JSON(info)
		}
	}
	return fiber.NewError(fiber.StatusNotFound, "session not found")
}

// handleServerStatus reports version, uptime, connection pool usage, and
// operation counters for the current connection's server.
func (d *deps) handleServerStatus(c fiber.Ctx) error {
	status, err := currentClient(c).ServerStatus(c.Context())
	if err != nil {
		return fmt.Errorf("server status: %w", err)
	}
	return c.JSON(status)
}

// redactURI strips the password from a mongodb:// connection string before
// it is ever stored or sent back to the frontend.
func redactURI(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		if username := u.User.Username(); username != "" {
			u.User = url.User(username)
		} else {
			u.User = nil
		}
	}
	return u.String()
}

// handleListBookmarks lists saved connection bookmarks (name and redacted
// URI only — never the raw file, which may embed credentials).
func (d *deps) handleListBookmarks(c fiber.Ctx) error {
	if d.bookmarksDir == "" {
		return c.JSON([]fiber.Map{})
	}
	saved, err := bookmarks.List(d.bookmarksDir)
	if err != nil {
		return fmt.Errorf("list bookmarks: %w", err)
	}
	out := make([]fiber.Map, len(saved))
	for i, b := range saved {
		out[i] = fiber.Map{"name": b.Name, "uri": redactURI(b.URL)}
	}
	return c.JSON(out)
}
