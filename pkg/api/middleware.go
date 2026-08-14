package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/client"
	"github.com/laubstein/gonosequel/pkg/session"
)

// sessionIDHeader is how the frontend selects which connection a request
// applies to in --sessions mode. In single-connection mode it is unused
// and every request resolves to session.DefaultID.
const sessionIDHeader = "X-Session-Id"

type localKey string

const clientLocalKey localKey = "client"
const sessionIDLocalKey localKey = "sessionID"

// withSession resolves the request's session ID to a connected client and
// stores both in locals for handlers to read via currentClient and
// currentSessionID. Requests with no matching session get 400, so a
// handler never has to nil-check its client.
func withSession(d *deps) fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get(sessionIDHeader)
		if id == "" {
			id = session.DefaultID
		}

		cl, err := d.registry.Get(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "no active connection for this session; connect first")
		}

		fiber.Locals[*client.Client](c, clientLocalKey, cl)
		fiber.Locals[string](c, sessionIDLocalKey, id)
		return c.Next()
	}
}

func currentClient(c fiber.Ctx) *client.Client {
	return fiber.Locals[*client.Client](c, clientLocalKey)
}

func currentSessionID(c fiber.Ctx) string {
	return fiber.Locals[string](c, sessionIDLocalKey)
}
