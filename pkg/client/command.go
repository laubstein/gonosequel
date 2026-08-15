package client

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// RunCommand is unsupported: MongoDB has no equivalent to a redis-cli-style
// raw command console in this UI (mongosh's own shell syntax is not a
// wire-level command the way Redis's is).
func (c *Client) RunCommand(ctx context.Context, dbName string, args []string) (any, error) {
	return nil, fmt.Errorf("run command: %w", driver.ErrUnsupported)
}
