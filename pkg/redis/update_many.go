package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// UpdateMany is unsupported: Redis has no concept of a filter matching
// multiple keys at once the way a MongoDB query does — RunCommand's
// SCAN-then-per-key-write is the closest equivalent, and that's a
// client-side loop the caller drives itself, not a single backend call.
func (c *Client) UpdateMany(ctx context.Context, dbName, collName string, filter, update driver.Doc) (matched, modified int64, err error) {
	return 0, 0, fmt.Errorf("update many: %w", driver.ErrUnsupported)
}
