package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// InferSchema samples up to sampleSize keys from collName and reports the
// distribution of Redis types found (string/hash/list/set/zset), plus the
// field names seen across sampled hashes — the closest Redis equivalent to
// the Mongo driver's $sample-based field-type inference, since Redis has
// no declared schema either. This is "what's in this collection", not a
// field-level type breakdown like Mongo's, because Redis values aren't
// documents with named fields (except hashes).
func (c *Client) InferSchema(ctx context.Context, dbName, collName string, sampleSize int64) ([]driver.SchemaField, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return nil, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return nil, err
	}
	if sampleSize <= 0 {
		sampleSize = driver.DefaultSchemaSampleSize
	}

	typeCounts := map[string]int{}
	fieldCounts := map[string]int{}
	var sampled int64

	if err := scanKeys(ctx, rc, "*", func(key string) error {
		if sampled >= sampleSize {
			return nil
		}
		if collectionOf(key) != collName {
			return nil
		}
		t, err := rc.Type(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("type %q: %w", key, err)
		}
		typeCounts[t]++
		sampled++
		if t == "hash" {
			fields, err := rc.HKeys(ctx, key).Result()
			if err == nil {
				for _, f := range fields {
					fieldCounts[f]++
				}
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("infer schema %q: %w", collName, err)
	}

	fields := make([]driver.SchemaField, 0, 1+len(fieldCounts))
	types := make([]driver.FieldType, 0, len(typeCounts))
	for t, n := range typeCounts {
		types = append(types, driver.FieldType{Type: t, Count: n})
	}
	fields = append(fields, driver.SchemaField{Path: "(key type)", Types: types})
	for f, n := range fieldCounts {
		fields = append(fields, driver.SchemaField{
			Path:  "hash field: " + f,
			Types: []driver.FieldType{{Type: "present", Count: n}},
		})
	}
	return fields, nil
}
