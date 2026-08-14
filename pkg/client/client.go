package client

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Sentinel errors returned by client operations, checked with errors.Is.
var (
	// ErrNotFound is returned when a requested database, collection, or
	// document does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when creating a collection or index
	// that already exists under that name.
	ErrAlreadyExists = errors.New("already exists")
)

// Client wraps a *mongo.Client, providing the operations the API layer
// needs without exposing driver types to callers outside this package.
type Client struct {
	mongo *mongo.Client
}

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
