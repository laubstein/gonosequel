package redis

import (
	"context"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// CollectionsOverview reuses ListCollections + Stats, same cost caveats as
// both.
func (c *Client) CollectionsOverview(ctx context.Context, dbName string) ([]driver.CollectionStats, error) {
	colls, err := c.ListCollections(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("collections overview: %w", err)
	}
	out := make([]driver.CollectionStats, 0, len(colls))
	for _, coll := range colls {
		stats, err := c.Stats(ctx, dbName, coll.Name)
		if err != nil {
			return nil, fmt.Errorf("collections overview: stats for %q: %w", coll.Name, err)
		}
		out = append(out, stats)
	}
	return out, nil
}

// IndexUsage always returns an empty list: Redis has no indexes to report
// usage for (see indexes.go).
func (c *Client) IndexUsage(ctx context.Context, dbName string) ([]driver.IndexUsageStat, error) {
	return nil, nil
}

// CurrentOps uses SLOWLOG GET as the closest available proxy: Redis is
// single-threaded and almost every command completes well under a second,
// so there is no real equivalent of MongoDB's currentOp (which shows
// queries actively in flight) — this will be empty most of the time, which
// is an accurate reflection of a healthy Redis server, not a bug. minSecs
// is only used to note in Description whether a slow command exceeded it.
func (c *Client) CurrentOps(ctx context.Context, minSecs int64) ([]driver.CurrentOp, error) {
	rc, err := c.conn(0)
	if err != nil {
		return nil, err
	}
	entries, err := rc.SlowLogGet(ctx, 25).Result()
	if err != nil {
		return nil, fmt.Errorf("slowlog get: %w", err)
	}

	out := make([]driver.CurrentOp, 0, len(entries))
	for _, e := range entries {
		secs := e.Duration.Seconds()
		if int64(secs) < minSecs {
			continue
		}
		op := ""
		if len(e.Args) > 0 {
			op = e.Args[0]
		}
		out = append(out, driver.CurrentOp{
			OpID:        e.ID,
			Namespace:   "",
			Op:          op,
			SecsRunning: int64(secs),
			Client:      e.ClientAddr,
			Description: fmt.Sprintf("%v", e.Args),
		})
	}
	return out, nil
}
