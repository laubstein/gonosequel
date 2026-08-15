// Package redis is the Redis/Valkey implementation of pkg/driver.Driver —
// the two are wire-compatible (RESP), so one implementation serves both
// --driver values. Redis has no named databases, collections, or JSON
// documents; see the doc comments on individual methods for how each
// driver.Driver capability maps onto Redis's actual data model, and where
// that mapping is lossy or simply unsupported.
package redis

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// collKeyDelim separates a key's collection prefix from the rest of it
// (e.g. "user:123" is in collection "user"). Keys with no delimiter fall
// into noPrefixCollection.
const collKeyDelim = ":"

// noPrefixCollection is the synthetic collection name for keys with no
// collKeyDelim in them at all.
const noPrefixCollection = "(no prefix)"

// Client is a connection to a Redis/Valkey server. Unlike MongoDB, where
// one connection can address every database, a go-redis client is bound to
// a single numbered database at construction time — Client lazily opens
// one goredis.Client per database index the caller actually touches, all
// sharing the same address/credentials.
type Client struct {
	base goredis.Options // template; Base.DB is overridden per index

	mu  sync.Mutex
	dbs map[int]*goredis.Client
}

var _ driver.Driver = (*Client)(nil)

// Connect parses a redis:// URI and verifies connectivity against the
// database it names (or 0, if none). Callers must call Close when done.
func Connect(ctx context.Context, uri string) (*Client, error) {
	opts, err := goredis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	c := &Client{base: *opts, dbs: map[int]*goredis.Client{}}
	rc, err := c.conn(opts.DB)
	if err != nil {
		return nil, err
	}
	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return c, nil
}

// conn returns the cached goredis.Client for dbIndex, opening one if this
// is the first time it's used.
func (c *Client) conn(dbIndex int) (*goredis.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rc, ok := c.dbs[dbIndex]; ok {
		return rc, nil
	}
	optsCopy := c.base
	optsCopy.DB = dbIndex
	rc := goredis.NewClient(&optsCopy)
	c.dbs[dbIndex] = rc
	return rc, nil
}

// Close disconnects every per-database client opened so far.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	for rc := range maps.Values(c.dbs) {
		if err := rc.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Capabilities reports the driver.Driver capabilities Redis actually
// supports: no aggregation pipeline, no query planner, and no secondary
// indexes without the RediSearch module (not attempted here).
func (c *Client) Capabilities() []string {
	return []string{driver.CapFind, driver.CapSchema, driver.CapTools, driver.CapCommand}
}

// dbIndex parses "database" name as Redis's numbered database index (0-15
// by default server config) — Redis has no named databases, so
// driver.Driver's dbName string is always a base-10 integer here.
func dbIndex(dbName string) (int, error) {
	n, err := strconv.Atoi(dbName)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid Redis database index: %w", dbName, err)
	}
	return n, nil
}
