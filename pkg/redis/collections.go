package redis

import (
	"context"
	"fmt"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// scanKeys walks every key in rc via SCAN (never KEYS *, which blocks the
// server), calling visit for each one. SCAN's cursor guarantees every key
// present for the whole scan is seen at least once, but gives no
// consistent snapshot — keys written or deleted mid-scan may or may not
// appear. That's an accepted tradeoff here, the same one every Redis GUI
// makes, in exchange for not blocking the server on a large keyspace.
func scanKeys(ctx context.Context, rc *goredis.Client, match string, visit func(key string) error) error {
	var cursor uint64
	for {
		keys, next, err := rc.Scan(ctx, cursor, match, 1000).Result()
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		for _, k := range keys {
			if err := visit(k); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// collectionOf returns the synthetic collection name for a Redis key: the
// part before the first collKeyDelim, or noPrefixCollection if there is
// none.
func collectionOf(key string) string {
	prefix, _, ok := strings.Cut(key, collKeyDelim)
	if !ok {
		return noPrefixCollection
	}
	return prefix
}

// ListCollections derives the collection list by scanning every key in the
// database and grouping by collectionOf. This is O(number of keys) per
// call — there is no cheaper way to enumerate distinct key prefixes in
// Redis without maintaining a secondary index ourselves, which this driver
// deliberately doesn't do.
func (c *Client) ListCollections(ctx context.Context, dbName string) ([]driver.CollectionInfo, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return nil, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	if err := scanKeys(ctx, rc, "*", func(key string) error {
		seen[collectionOf(key)] = struct{}{}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}

	out := make([]driver.CollectionInfo, 0, len(seen))
	for name := range seen {
		out = append(out, driver.CollectionInfo{Name: name, Type: "collection"})
	}
	return out, nil
}

// CreateCollection always fails: a Redis "collection" is just a shared key
// prefix, so it only exists once a key with that prefix exists. There is
// no way to create an empty one, unlike a MongoDB collection.
func (c *Client) CreateCollection(ctx context.Context, dbName, collName string, opts driver.CreateCollectionOptions) error {
	return fmt.Errorf("create collection %q: %w (create a key named %q:... instead)", collName, driver.ErrUnsupported, collName)
}

// DropCollection deletes every key whose collectionOf is collName.
func (c *Client) DropCollection(ctx context.Context, dbName, collName string) error {
	idx, err := dbIndex(dbName)
	if err != nil {
		return err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return err
	}

	var keys []string
	if err := scanKeys(ctx, rc, "*", func(key string) error {
		if collectionOf(key) == collName {
			keys = append(keys, key)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("drop collection %q: %w", collName, err)
	}
	if len(keys) == 0 {
		return driver.ErrNotFound
	}
	if err := rc.Unlink(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("drop collection %q: %w", collName, err)
	}
	return nil
}

// RenameCollection renames every key in oldName's group by swapping its
// prefix for newName, one RENAME per key — there is no bulk rename in
// Redis. Expensive for large collections; documented, not optimized here.
func (c *Client) RenameCollection(ctx context.Context, dbName, oldName, newName string) error {
	idx, err := dbIndex(dbName)
	if err != nil {
		return err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return err
	}

	var keys []string
	if err := scanKeys(ctx, rc, "*", func(key string) error {
		if collectionOf(key) == oldName {
			keys = append(keys, key)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("rename collection %q: %w", oldName, err)
	}
	if len(keys) == 0 {
		return driver.ErrNotFound
	}
	for _, key := range keys {
		_, rest, ok := strings.Cut(key, collKeyDelim)
		newKey := newName
		if ok {
			newKey = newName + collKeyDelim + rest
		}
		if err := rc.Rename(ctx, key, newKey).Err(); err != nil {
			return fmt.Errorf("rename key %q -> %q: %w", key, newKey, err)
		}
	}
	return nil
}

// Stats reports the key count for collName exactly (via the same scan as
// ListCollections/DropCollection) and an estimated total size in bytes,
// sampled with MEMORY USAGE over at most sampleStatsKeys keys and
// extrapolated — an exact sum would mean calling MEMORY USAGE once per
// key, too expensive for a large collection.
func (c *Client) Stats(ctx context.Context, dbName, collName string) (driver.CollectionStats, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return driver.CollectionStats{}, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return driver.CollectionStats{}, err
	}

	const sampleStatsKeys = 100
	var (
		keys        []string
		sampledSize int64
		sampled     int
	)
	if err := scanKeys(ctx, rc, "*", func(key string) error {
		if collectionOf(key) != collName {
			return nil
		}
		keys = append(keys, key)
		if sampled < sampleStatsKeys {
			if n, err := rc.MemoryUsage(ctx, key).Result(); err == nil {
				sampledSize += n
				sampled++
			}
		}
		return nil
	}); err != nil {
		return driver.CollectionStats{}, fmt.Errorf("stats %q: %w", collName, err)
	}
	if len(keys) == 0 {
		return driver.CollectionStats{}, driver.ErrNotFound
	}

	avg := int64(0)
	if sampled > 0 {
		avg = sampledSize / int64(sampled)
	}
	return driver.CollectionStats{
		Name:       collName,
		Count:      int64(len(keys)),
		SizeBytes:  avg * int64(len(keys)),
		AvgObjSize: avg,
	}, nil
}
