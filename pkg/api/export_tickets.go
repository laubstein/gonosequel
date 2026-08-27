package api

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// exportTicketTTL is how long a ticket stays claimable. It only has to
	// cover the gap between the frontend receiving the token and the
	// browser starting the download, which is one round trip.
	exportTicketTTL = 30 * time.Second

	// maxExportTickets caps the store so an authenticated client can't grow
	// it without bound by requesting tickets it never redeems.
	maxExportTickets = 256
)

// exportTicket is a one-shot, short-lived authorization to download one
// specific export without an X-Session-Id header — a browser download
// navigation cannot send one, which is why /export is otherwise
// unreachable in --sessions mode.
//
// The query text is stored exactly as the issuing request carried it, so
// parseFindOptionsFrom stays the single place that interprets it.
type exportTicket struct {
	sessionID  string
	db         string
	coll       string
	format     string
	filter     string
	projection string
	sort       string
	expires    time.Time
}

// query exposes the ticket's stored query text with the same signature as
// fiber.Ctx.Query, so the download path can reuse parseFindOptionsFrom.
func (t exportTicket) query(key string, def ...string) string {
	var v string
	switch key {
	case "filter":
		v = t.filter
	case "projection":
		v = t.projection
	case "sort":
		v = t.sort
	}
	if v == "" && len(def) > 0 {
		return def[0]
	}
	return v
}

// exportTicketStore holds unredeemed export tickets. It is in-memory and
// therefore single-process, the same assumption history.Store already
// makes — gonosequel is a single binary, not a clustered service.
type exportTicketStore struct {
	mu sync.Mutex
	m  map[string]exportTicket
}

func newExportTicketStore() *exportTicketStore {
	return &exportTicketStore{m: make(map[string]exportTicket)}
}

// issue registers t and returns its opaque token. Expired entries are
// swept here rather than on a timer, so the store needs no goroutine and
// no shutdown hook.
func (s *exportTicketStore) issue(t exportTicket) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for token, existing := range s.m {
		if now.After(existing.expires) {
			delete(s.m, token)
		}
	}
	if len(s.m) >= maxExportTickets {
		return "", false
	}

	token := uuid.NewString()
	s.m[token] = t
	return token, true
}

// claim returns the ticket for token and removes it. A token is valid for
// exactly one download, so one that later reaches browser history or a
// proxy's access log has already been spent.
func (s *exportTicketStore) claim(token string) (exportTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.m[token]
	if !ok {
		return exportTicket{}, false
	}
	delete(s.m, token)
	if time.Now().After(t.expires) {
		return exportTicket{}, false
	}
	return t, true
}
