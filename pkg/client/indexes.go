package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// ListIndexes returns every index on a collection.
func (c *Client) ListIndexes(ctx context.Context, dbName, collName string) ([]driver.IndexInfo, error) {
	cur, err := c.collection(dbName, collName).Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list indexes %q.%q: %w", dbName, collName, err)
	}
	defer cur.Close(ctx)

	var out []driver.IndexInfo
	for cur.Next(ctx) {
		var raw struct {
			Name                    string `bson:"name"`
			Key                     bson.D `bson:"key"`
			Unique                  bool   `bson:"unique"`
			Sparse                  bool   `bson:"sparse"`
			ExpireAfterSeconds      *int64 `bson:"expireAfterSeconds"`
			PartialFilterExpression bson.D `bson:"partialFilterExpression"`
		}
		if err := cur.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode index: %w", err)
		}

		var partialFilter driver.Doc
		if len(raw.PartialFilterExpression) > 0 {
			if d, ok := toAny(raw.PartialFilterExpression).(driver.Doc); ok {
				partialFilter = d
			}
		}

		out = append(out, driver.IndexInfo{
			Name:                    raw.Name,
			Keys:                    toOrderedDoc(raw.Key),
			Unique:                  raw.Unique,
			Sparse:                  raw.Sparse,
			ExpireAfterSeconds:      raw.ExpireAfterSeconds,
			PartialFilterExpression: partialFilter,
		})
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexes: %w", err)
	}
	return out, nil
}

// CreateIndex creates an index from the given key spec (field -> 1 | -1)
// and returns its generated name.
func (c *Client) CreateIndex(ctx context.Context, dbName, collName string, keys driver.OrderedDoc, opts driver.CreateIndexOptions) (string, error) {
	model := mongo.IndexModel{Keys: toBSOND(keys)}
	idxOpts := options.Index()
	changed := false
	if opts.Unique {
		idxOpts.SetUnique(true)
		changed = true
	}
	if opts.Sparse {
		idxOpts.SetSparse(true)
		changed = true
	}
	if opts.ExpireAfterSeconds != nil {
		idxOpts.SetExpireAfterSeconds(int32(*opts.ExpireAfterSeconds))
		changed = true
	}
	if opts.PartialFilterExpression != nil {
		idxOpts.SetPartialFilterExpression(toBSON(opts.PartialFilterExpression))
		changed = true
	}
	if changed {
		model.Options = idxOpts
	}

	name, err := c.collection(dbName, collName).Indexes().CreateOne(ctx, model)
	if err != nil {
		return "", fmt.Errorf("create index on %q.%q: %w", dbName, collName, err)
	}
	return name, nil
}

// DropIndex deletes an index by name.
func (c *Client) DropIndex(ctx context.Context, dbName, collName, name string) error {
	err := c.collection(dbName, collName).Indexes().DropOne(ctx, name)
	if err != nil {
		return fmt.Errorf("drop index %q on %q.%q: %w", name, dbName, collName, err)
	}
	return nil
}

// UpdateIndexTTL changes an existing TTL index's expireAfterSeconds via
// collMod — the one index property MongoDB allows changing without
// dropping and recreating the index. Identifies the index by name, not
// key pattern, so it works regardless of what the index's keys are.
func (c *Client) UpdateIndexTTL(ctx context.Context, dbName, collName, indexName string, expireAfterSeconds int64) error {
	cmd := bson.D{
		{Key: "collMod", Value: collName},
		{Key: "index", Value: bson.D{
			{Key: "name", Value: indexName},
			{Key: "expireAfterSeconds", Value: expireAfterSeconds},
		}},
	}
	if err := c.mongo.Database(dbName).RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("update TTL for index %q on %q.%q: %w", indexName, dbName, collName, err)
	}
	return nil
}
