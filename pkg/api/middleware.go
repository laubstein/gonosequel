package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/session"
)

// sessionIDHeader is how the frontend selects which connection a request
// applies to in --sessions mode. In single-connection mode it is unused
// and every request resolves to session.DefaultID.
const sessionIDHeader = "X-Session-Id"

type localKey string

const clientLocalKey localKey = "client"
const sessionIDLocalKey localKey = "sessionID"

// resolveSessionID extracts the session ID a request refers to. With no
// header, it's the internal single-connection fallback (never signed, so
// skipping verification for it is safe). With a header and no
// d.sessionSecret configured, the header is used as-is (today's unsigned
// behavior, preserved for compatibility). With both a header and a secret
// configured, the header must be a valid token from session.SignID — the
// raw id it was generated for is returned only if the signature checks
// out, so a client can't forge or guess another session's ID.
func resolveSessionID(d *deps, c fiber.Ctx) (string, bool) {
	raw := c.Get(sessionIDHeader)
	if raw == "" {
		return session.DefaultID, true
	}
	if d.sessionSecret == "" {
		return raw, true
	}
	return session.VerifyID(d.sessionSecret, raw)
}

// withSession resolves the request's session ID to a connected client and
// stores both in locals for handlers to read via currentClient and
// currentSessionID. Requests with no matching session get 400, so a
// handler never has to nil-check its client.
func withSession(d *deps) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, ok := resolveSessionID(d, c)
		if !ok {
			return fiber.NewError(fiber.StatusBadRequest, "invalid session id")
		}

		cl, err := d.registry.Get(id)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "no active connection for this session; connect first")
		}

		// Per-session read-only, independent of (and in addition to) the
		// server-wide --readonly flag already enforced by rejectWrites in
		// New: a user can opt an individual --sessions connection into
		// read-only from the connect form even when the server default is
		// read-write. handleConnect forces Info.Readonly true whenever the
		// server-wide flag is set, so this check alone is enough — it
		// never needs to also consult d.readonly here.
		if info, err := d.registry.Info(id); err == nil && info.Readonly {
			if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
				return fiber.NewError(fiber.StatusForbidden, "this connection is read-only")
			}
		}

		fiber.Locals[driver.Driver](c, clientLocalKey, cl)
		fiber.Locals[string](c, sessionIDLocalKey, id)
		return c.Next()
	}
}

func currentClient(c fiber.Ctx) driver.Driver {
	return fiber.Locals[driver.Driver](c, clientLocalKey)
}

func currentSessionID(c fiber.Ctx) string {
	return fiber.Locals[string](c, sessionIDLocalKey)
}
