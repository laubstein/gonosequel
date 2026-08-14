package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// ListCollections returns every collection (and view) in a database.
func (c *Client) ListCollections(ctx context.Context, dbName string) ([]driver.CollectionInfo, error) {
	cur, err := c.mongo.Database(dbName).ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("list collections in %q: %w", dbName, err)
	}
	defer cur.Close(ctx)

	var out []driver.CollectionInfo
	for cur.Next(ctx) {
		var raw struct {
			Name string `bson:"name"`
			Type string `bson:"type"`
		}
		if err := cur.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode collection info: %w", err)
		}
		out = append(out, driver.CollectionInfo{Name: raw.Name, Type: raw.Type})
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate collections: %w", err)
	}
	return out, nil
}

// CreateCollection creates a collection with the given options.
func (c *Client) CreateCollection(ctx context.Context, dbName, collName string, opts driver.CreateCollectionOptions) error {
	createOpts := options.CreateCollection()
	if opts.Capped {
		createOpts.SetCapped(true).SetSizeInBytes(opts.MaxSizeByte)
		if opts.MaxDocs > 0 {
			createOpts.SetMaxDocuments(opts.MaxDocs)
		}
	}
	if err := c.mongo.Database(dbName).CreateCollection(ctx, collName, createOpts); err != nil {
		return fmt.Errorf("create collection %q.%q: %w", dbName, collName, err)
	}
	return nil
}

// DropCollection deletes a collection and all its documents and indexes.
func (c *Client) DropCollection(ctx context.Context, dbName, collName string) error {
	if err := c.mongo.Database(dbName).Collection(collName).Drop(ctx); err != nil {
		return fmt.Errorf("drop collection %q.%q: %w", dbName, collName, err)
	}
	return nil
}

// RenameCollection renames a collection within the same database.
func (c *Client) RenameCollection(ctx context.Context, dbName, oldName, newName string) error {
	admin := c.mongo.Database("admin")
	cmd := bson.D{
		{Key: "renameCollection", Value: dbName + "." + oldName},
		{Key: "to", Value: dbName + "." + newName},
	}
	if err := admin.RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("rename collection %q.%q -> %q: %w", dbName, oldName, newName, err)
	}
	return nil
}

// Stats reports size and count metrics for a collection.
func (c *Client) Stats(ctx context.Context, dbName, collName string) (driver.CollectionStats, error) {
	var raw struct {
		Count      int64 `bson:"count"`
		Size       int64 `bson:"size"`
		StorageSz  int64 `bson:"storageSize"`
		AvgObjSize int64 `bson:"avgObjSize"`
		NIndexes   int64 `bson:"nindexes"`
		TotalIdxSz int64 `bson:"totalIndexSize"`
	}
	cmd := bson.D{{Key: "collStats", Value: collName}}
	if err := c.mongo.Database(dbName).RunCommand(ctx, cmd).Decode(&raw); err != nil {
		return driver.CollectionStats{}, fmt.Errorf("collStats %q.%q: %w", dbName, collName, err)
	}
	return driver.CollectionStats{
		Count:        raw.Count,
		SizeBytes:    raw.Size,
		StorageBytes: raw.StorageSz,
		IndexBytes:   raw.TotalIdxSz,
		AvgObjSize:   raw.AvgObjSize,
		IndexCount:   raw.NIndexes,
	}, nil
}

// collection is a small helper to reduce repetition in the other files.
func (c *Client) collection(dbName, collName string) *mongo.Collection {
	return c.mongo.Database(dbName).Collection(collName)
}
