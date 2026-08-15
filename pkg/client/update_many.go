package client

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// UpdateMany applies update (an update document, e.g. {"$set": {...}}, not
// a full replacement document) to every document matching filter — the
// same semantics as mongosh's own db.collection.updateMany.
func (c *Client) UpdateMany(ctx context.Context, dbName, collName string, filter, update driver.Doc) (matched, modified int64, err error) {
	res, err := c.collection(dbName, collName).UpdateMany(ctx, toBSON(filter), toBSON(update))
	if err != nil {
		return 0, 0, fmt.Errorf("update many %q.%q: %w", dbName, collName, err)
	}
	return res.MatchedCount, res.ModifiedCount, nil
}
