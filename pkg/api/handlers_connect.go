package api

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/laubstein/mongo-express-go/pkg/client"
	"github.com/laubstein/mongo-express-go/pkg/session"
)

type connectRequest struct {
	URL  string `json:"url"`
	Name string `json:"name"`
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
	if req.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url is required")
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	cl, err := client.Connect(ctx, req.URL)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("connect: %v", err))
	}

	id := session.DefaultID
	if d.sessions {
		id = uuid.NewString()
	}

	name := req.Name
	if name == "" {
		name = redactURI(req.URL)
	}

	d.registry.Put(id, cl, session.Info{ID: id, URI: redactURI(req.URL), Name: name})

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

// handleServerStatus reports a trimmed serverStatus summary for the
// current connection.
func (d *deps) handleServerStatus(c fiber.Ctx) error {
	version, err := currentClient(c).ServerVersion(c.Context())
	if err != nil {
		return fmt.Errorf("server status: %w", err)
	}
	return c.JSON(fiber.Map{"version": version})
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
