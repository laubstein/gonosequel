package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Explain is unsupported: Redis has no query planner to report on — every
// command is O(1) or O(n) in the size of the value it touches, not a plan
// chosen from alternatives.
func (c *Client) Explain(ctx context.Context, dbName, collName string, filter driver.Doc) (driver.Doc, error) {
	return nil, fmt.Errorf("explain: %w (Redis has no query planner)", driver.ErrUnsupported)
}
