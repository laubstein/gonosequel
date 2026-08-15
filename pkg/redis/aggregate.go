package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Aggregate is unsupported: Redis has no multi-stage aggregation pipeline
// concept.
func (c *Client) Aggregate(ctx context.Context, dbName, collName string, pipeline []driver.Doc) ([]driver.Doc, error) {
	return nil, fmt.Errorf("aggregate: %w (Redis has no aggregation pipeline)", driver.ErrUnsupported)
}
