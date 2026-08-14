// Package client is the MongoDB implementation of pkg/driver.Driver: it
// wraps the mongo-driver, converting between BSON and the generic
// pkg/driver types at every exported method's boundary so no caller
// outside this package needs to know MongoDB is behind the connection.
package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Client wraps a *mongo.Client, providing the operations the API layer
// needs without exposing driver types to callers outside this package.
type Client struct {
	mongo *mongo.Client
}

// var _ documents, at compile time, that *Client satisfies driver.Driver.
var _ driver.Driver = (*Client)(nil)

// Connect dials the given MongoDB URI and verifies connectivity with a
// ping. Callers must call Close when done.
func Connect(ctx context.Context, uri string) (*Client, error) {
	mc, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		_ = mc.Disconnect(ctx)
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Client{mongo: mc}, nil
}

// Close disconnects the underlying MongoDB client.
func (c *Client) Close(ctx context.Context) error {
	return c.mongo.Disconnect(ctx)
}
