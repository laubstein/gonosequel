// Package history keeps an in-memory, per-session log of executed queries.
// It is intentionally not persisted: history is a convenience for the
// current server run, not an audit trail.
package history

import "sync"

// Entry is one executed query.
type Entry struct {
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"`
	At         string `json:"at"`
}

// MaxEntriesPerSession bounds memory use: older entries are dropped once
// this many are recorded for a session.
const MaxEntriesPerSession = 200

// Store holds query history per session ID, safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	entries map[string][]Entry
}

// NewStore returns an empty history store.
func NewStore() *Store {
	return &Store{entries: map[string][]Entry{}}
}

// Add appends an entry to the session's history, trimming the oldest
// entries if MaxEntriesPerSession is exceeded.
func (s *Store) Add(sessionID string, e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := append(s.entries[sessionID], e)
	if len(entries) > MaxEntriesPerSession {
		entries = entries[len(entries)-MaxEntriesPerSession:]
	}
	s.entries[sessionID] = entries
}

// List returns the session's history, most recent first.
func (s *Store) List(sessionID string) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	src := s.entries[sessionID]
	out := make([]Entry, len(src))
	for i := range src {
		out[i] = src[len(src)-1-i]
	}
	return out
}
