package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Find scans every key in collName, optionally narrowed by a
// "$keyPattern" glob in opts.Filter (the only filter Redis supports —
// there is no query language over values), sorts the matches for a stable
// page order, and slices skip/limit over that sorted list. Sort and
// Projection are ignored: they have no meaning over opaque key/value
// pairs.
func (c *Client) Find(ctx context.Context, dbName, collName string, opts driver.FindOptions) (driver.FindResult, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return driver.FindResult{}, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return driver.FindResult{}, err
	}

	var keyGlob string
	if opts.Filter != nil {
		if v, ok := opts.Filter["$keyPattern"].(string); ok {
			keyGlob = v
		}
	}

	var keys []string
	if err := scanKeys(ctx, rc, "*", func(key string) error {
		if collectionOf(key) != collName {
			return nil
		}
		if keyGlob != "" {
			if ok, _ := path.Match(keyGlob, key); !ok {
				return nil
			}
		}
		keys = append(keys, key)
		return nil
	}); err != nil {
		return driver.FindResult{}, fmt.Errorf("find in %q: %w", collName, err)
	}
	sort.Strings(keys)

	total := int64(len(keys))
	start := min(opts.Skip, total)
	end := total
	if opts.Limit > 0 {
		end = min(start+opts.Limit, total)
	}

	docs := make([]driver.Doc, 0, end-start)
	for _, key := range keys[start:end] {
		doc, err := readKeyDoc(ctx, rc, key)
		if err != nil {
			return driver.FindResult{}, fmt.Errorf("find in %q: %w", collName, err)
		}
		docs = append(docs, doc)
	}

	return driver.FindResult{Documents: docs, Total: total, TotalIsEstimate: false}, nil
}

// FindOne fetches a single key, given as id (must be a string — the key
// itself is the id in Redis, there is no separate identity).
func (c *Client) FindOne(ctx context.Context, dbName, collName string, id any) (driver.Doc, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return nil, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return nil, err
	}
	key, ok := id.(string)
	if !ok {
		return nil, fmt.Errorf("redis document id must be a string key, got %T", id)
	}
	return readKeyDoc(ctx, rc, key)
}

// readKeyDoc builds the synthetic document {_id, type, ttl, value} the
// frontend's per-type Redis editor consumes. value's shape depends on
// type: a plain string for "string", a field->value map for "hash", an
// ordered array for "list", an unordered array for "set", and an array of
// {member, score} for "zset".
func readKeyDoc(ctx context.Context, rc *goredis.Client, key string) (driver.Doc, error) {
	t, err := rc.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("type %q: %w", key, err)
	}
	if t == "none" {
		return nil, driver.ErrNotFound
	}

	ttlSecs := int64(-1)
	if ttl, err := rc.TTL(ctx, key).Result(); err == nil && ttl > 0 {
		ttlSecs = int64(ttl.Seconds())
	}

	var value any
	switch t {
	case "string":
		v, err := rc.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("get %q: %w", key, err)
		}
		value = v
	case "hash":
		m, err := rc.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("hgetall %q: %w", key, err)
		}
		doc := make(driver.Doc, len(m))
		for k, v := range m {
			doc[k] = v
		}
		value = doc
	case "list":
		l, err := rc.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("lrange %q: %w", key, err)
		}
		value = l
	case "set":
		s, err := rc.SMembers(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("smembers %q: %w", key, err)
		}
		value = s
	case "zset":
		zs, err := rc.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("zrange %q: %w", key, err)
		}
		members := make([]driver.Doc, len(zs))
		for i, z := range zs {
			members[i] = driver.Doc{"member": z.Member, "score": z.Score}
		}
		value = members
	default:
		return nil, fmt.Errorf("redis type %q for key %q: %w", t, key, driver.ErrUnsupported)
	}

	return driver.Doc{"_id": key, "type": t, "ttl": ttlSecs, "value": value}, nil
}

// InsertOne writes doc as a new Redis key. doc must have "type" (one of
// string/hash/list/set/zset) and "value" shaped as readKeyDoc documents;
// "_id", if given, is the key name — otherwise one is generated under
// collName.
func (c *Client) InsertOne(ctx context.Context, dbName, collName string, doc driver.Doc) (any, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return nil, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return nil, err
	}

	key, _ := doc["_id"].(string)
	if key == "" {
		key = collName + collKeyDelim + randomID()
	}
	if err := writeKeyDoc(ctx, rc, key, doc); err != nil {
		return nil, fmt.Errorf("insert %q: %w", key, err)
	}
	return key, nil
}

// ReplaceOne overwrites the key named by id (must be a string) with doc,
// deleting whatever was there first — necessary since the new value may
// be a different Redis type than the old one.
func (c *Client) ReplaceOne(ctx context.Context, dbName, collName string, id any, doc driver.Doc) error {
	idx, err := dbIndex(dbName)
	if err != nil {
		return err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return err
	}
	key, ok := id.(string)
	if !ok {
		return fmt.Errorf("redis document id must be a string key, got %T", id)
	}
	n, err := rc.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("exists %q: %w", key, err)
	}
	if n == 0 {
		return driver.ErrNotFound
	}
	if err := rc.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("replace %q: %w", key, err)
	}
	if err := writeKeyDoc(ctx, rc, key, doc); err != nil {
		return fmt.Errorf("replace %q: %w", key, err)
	}
	return nil
}

// DeleteOne deletes the key named by id.
func (c *Client) DeleteOne(ctx context.Context, dbName, collName string, id any) error {
	idx, err := dbIndex(dbName)
	if err != nil {
		return err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return err
	}
	key, ok := id.(string)
	if !ok {
		return fmt.Errorf("redis document id must be a string key, got %T", id)
	}
	n, err := rc.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	if n == 0 {
		return driver.ErrNotFound
	}
	return nil
}

// writeKeyDoc dispatches to the right Redis write command based on
// doc["type"], as produced by readKeyDoc / the frontend's per-type editor,
// then applies doc["ttl"] if present.
func writeKeyDoc(ctx context.Context, rc *goredis.Client, key string, doc driver.Doc) error {
	if err := writeKeyValue(ctx, rc, key, doc); err != nil {
		return err
	}
	return applyKeyTTL(ctx, rc, key, doc)
}

// applyKeyTTL sets or clears the key's expiry from doc["ttl"], matching
// the field readKeyDoc reports: a positive number of seconds sets one,
// anything <= 0 (readKeyDoc's -1 for "no expiry") clears it. An absent
// ttl leaves the key alone — but note every write path here recreates the
// key, which drops its old expiry anyway, so an editor that never sends
// ttl silently makes keys permanent.
func applyKeyTTL(ctx context.Context, rc *goredis.Client, key string, doc driver.Doc) error {
	raw, ok := doc["ttl"]
	if !ok {
		return nil
	}
	secs, ok := toFloat(raw)
	if !ok {
		return fmt.Errorf("ttl must be a number of seconds, got %T", raw)
	}
	if secs <= 0 {
		if err := rc.Persist(ctx, key).Err(); err != nil {
			return fmt.Errorf("clear ttl on %q: %w", key, err)
		}
		return nil
	}
	if err := rc.Expire(ctx, key, time.Duration(secs)*time.Second).Err(); err != nil {
		return fmt.Errorf("set ttl on %q: %w", key, err)
	}
	return nil
}

func writeKeyValue(ctx context.Context, rc *goredis.Client, key string, doc driver.Doc) error {
	t, _ := doc["type"].(string)
	value := doc["value"]

	switch t {
	case "string":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("string value must be a string, got %T", value)
		}
		return rc.Set(ctx, key, s, 0).Err()
	case "hash":
		m, err := toStringMap(value)
		if err != nil {
			return err
		}
		if len(m) == 0 {
			return nil
		}
		return rc.HSet(ctx, key, m).Err()
	case "list":
		items, err := toStringSlice(value)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return rc.RPush(ctx, key, toAnySlice(items)...).Err()
	case "set":
		items, err := toStringSlice(value)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return rc.SAdd(ctx, key, toAnySlice(items)...).Err()
	case "zset":
		members, err := toZMembers(value)
		if err != nil {
			return err
		}
		if len(members) == 0 {
			return nil
		}
		return rc.ZAdd(ctx, key, members...).Err()
	default:
		return fmt.Errorf("unknown redis value type %q: %w", t, driver.ErrUnsupported)
	}
}

// toFloat accepts the several numeric shapes an Extended JSON round trip
// can produce for a plain number (int32/int64 wrappers decode to Go ints,
// a bare JSON number to float64).
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func toStringMap(v any) (map[string]string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("hash value must be an object, got %T", v)
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		out[k] = fmt.Sprint(val)
	}
	return out, nil
}

func toStringSlice(v any) ([]string, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("value must be an array, got %T", v)
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		out[i] = fmt.Sprint(item)
	}
	return out, nil
}

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func toZMembers(v any) ([]goredis.Z, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("zset value must be an array, got %T", v)
	}
	out := make([]goredis.Z, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("zset member must be an object with member/score, got %T", item)
		}
		score := 0.0
		switch s := m["score"].(type) {
		case float64:
			score = s
		case int:
			score = float64(s)
		}
		out = append(out, goredis.Z{Member: fmt.Sprint(m["member"]), Score: score})
	}
	return out, nil
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
