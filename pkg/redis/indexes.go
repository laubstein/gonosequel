package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// ListIndexes always returns an empty list: Redis has no secondary
// indexes without the RediSearch module, which this driver does not
// attempt to support.
func (c *Client) ListIndexes(ctx context.Context, dbName, collName string) ([]driver.IndexInfo, error) {
	return nil, nil
}

// CreateIndex is unsupported — see ListIndexes.
func (c *Client) CreateIndex(ctx context.Context, dbName, collName string, keys driver.OrderedDoc, opts driver.CreateIndexOptions) (string, error) {
	return "", fmt.Errorf("create index: %w (Redis has no secondary indexes without the RediSearch module)", driver.ErrUnsupported)
}

// DropIndex is unsupported — see ListIndexes.
func (c *Client) DropIndex(ctx context.Context, dbName, collName, name string) error {
	return fmt.Errorf("drop index: %w", driver.ErrUnsupported)
}

// UpdateIndexTTL is unsupported — see ListIndexes.
func (c *Client) UpdateIndexTTL(ctx context.Context, dbName, collName, indexName string, expireAfterSeconds int64) error {
	return fmt.Errorf("update index TTL: %w", driver.ErrUnsupported)
}
