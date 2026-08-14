package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DatabaseInfo summarizes a single database.
type DatabaseInfo struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
}

// ListDatabases returns every database on the server, with its on-disk
// size.
func (c *Client) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	result, err := c.mongo.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	dbs := make([]DatabaseInfo, 0, len(result.Databases))
	for _, d := range result.Databases {
		dbs = append(dbs, DatabaseInfo{Name: d.Name, SizeBytes: d.SizeOnDisk})
	}
	return dbs, nil
}

// CreateDatabase creates a database by creating an initial collection in
// it — MongoDB has no explicit "create database" command, since databases
// and collections are created lazily on first write.
func (c *Client) CreateDatabase(ctx context.Context, dbName, initialCollection string) error {
	if initialCollection == "" {
		initialCollection = "_init"
	}
	if err := c.mongo.Database(dbName).CreateCollection(ctx, initialCollection); err != nil {
		return fmt.Errorf("create database %q: %w", dbName, err)
	}
	return nil
}

// DropDatabase deletes a database and all its collections.
func (c *Client) DropDatabase(ctx context.Context, dbName string) error {
	if err := c.mongo.Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("drop database %q: %w", dbName, err)
	}
	return nil
}
