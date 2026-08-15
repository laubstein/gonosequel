// Package session tracks active database connections, keyed by session ID.
// In single-connection mode there is exactly one entry, created at
// startup; in --sessions mode, the API layer creates and destroys entries
// as users connect and disconnect from the UI.
package session

import (
	"context"
	"errors"
	"sync"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// ErrNotFound is returned when looking up a session ID that has no active
// connection.
var ErrNotFound = errors.New("session not found")

// DefaultID is the session ID used in single-connection mode, where the
// server was started with a fixed --url and there is no session picker in
// the UI.
const DefaultID = "default"

// Info describes an active session without exposing the underlying client.
type Info struct {
	ID   string `json:"id"`
	URI  string `json:"uri"` // credentials redacted
	Name string `json:"name"`
	// Driver names which backend this session is connected to (e.g.
	// "mongodb", "redis") — set by the caller of Put, since Registry has
	// no way to derive a name from a driver.Driver value alone. In
	// --sessions mode different sessions can be different backends, so
	// this is the per-session source of truth; Config.Driver on
	// pkg/api.New is only the startup default and can disagree.
	Driver string `json:"driver"`
	// Capabilities lists which driver.Cap* capabilities this connection's
	// backend actually supports (see driver.Driver.Capabilities), so the
	// frontend can hide UI for what it doesn't have instead of showing it
	// and failing on click. Snapshotted once at Put, since a connection's
	// backend doesn't change capabilities mid-session.
	Capabilities []string `json:"capabilities"`
	// Readonly marks this specific session as read-only, independent of
	// the server-wide --readonly flag: in --sessions mode a user can opt
	// a connection into read-only from the connect form even when the
	// server itself wasn't started with --readonly. The caller of Put
	// (handleConnect) is responsible for forcing this true whenever the
	// server-wide flag is set, so a session can never be *less*
	// restrictive than the server default regardless of what the client
	// requested.
	Readonly bool `json:"readonly"`
}

// Registry holds every active session, safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*entry
}

type entry struct {
	client driver.Driver
	info   Info
}

// NewRegistry returns an empty session registry.
func NewRegistry() *Registry {
	return &Registry{sessions: map[string]*entry{}}
}

// Put registers a connected driver under id, replacing any existing entry.
func (r *Registry) Put(id string, d driver.Driver, info Info) {
	info.Capabilities = d.Capabilities()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[id] = &entry{client: d, info: info}
}

// Get returns the driver registered under id.
func (r *Registry) Get(id string) (driver.Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return e.client, nil
}

// Info returns the info registered under id.
func (r *Registry) Info(id string) (Info, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.sessions[id]
	if !ok {
		return Info{}, ErrNotFound
	}
	return e.info, nil
}

// Remove closes and removes the session registered under id, if any.
func (r *Registry) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	e, ok := r.sessions[id]
	delete(r.sessions, id)
	r.mu.Unlock()

	if !ok {
		return nil
	}
	return e.client.Close(ctx)
}

// List returns info for every active session.
func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.sessions))
	for _, e := range r.sessions {
		out = append(out, e.info)
	}
	return out
}
