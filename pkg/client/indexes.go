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
			Name   string `bson:"name"`
			Key    bson.D `bson:"key"`
			Unique bool   `bson:"unique"`
		}
		if err := cur.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode index: %w", err)
		}
		out = append(out, driver.IndexInfo{Name: raw.Name, Keys: toOrderedDoc(raw.Key), Unique: raw.Unique})
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexes: %w", err)
	}
	return out, nil
}

// CreateIndex creates an index from the given key spec (field -> 1 | -1)
// and returns its generated name.
func (c *Client) CreateIndex(ctx context.Context, dbName, collName string, keys driver.OrderedDoc, unique bool) (string, error) {
	model := mongo.IndexModel{Keys: toBSOND(keys)}
	if unique {
		model.Options = options.Index().SetUnique(true)
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
