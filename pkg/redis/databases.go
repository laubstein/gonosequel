package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// maxDatabases is the classic Redis default database count (databases 0
// through 15). A server configured with a different `databases` directive
// may have more or fewer — ListDatabases still reports whatever INFO
// keyspace actually shows, this is only the fallback range used to also
// list empty databases below that.
const maxDatabases = 16

// ListDatabases reports every Redis database index with at least one key
// (via a single INFO keyspace call), padded out with the empty databases
// in 0..maxDatabases-1 so they're selectable even before they have data.
func (c *Client) ListDatabases(ctx context.Context) ([]driver.DatabaseInfo, error) {
	rc, err := c.conn(0)
	if err != nil {
		return nil, err
	}
	info, err := rc.Info(ctx, "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("info keyspace: %w", err)
	}

	counts := map[int]int64{}
	for _, line := range strings.Split(info, "\r\n") {
		// Lines look like "db0:keys=3,expires=0,avg_ttl=0,subexpiry=0"
		if !strings.HasPrefix(line, "db") {
			continue
		}
		idxStr, rest, ok := strings.Cut(strings.TrimPrefix(line, "db"), ":")
		if !ok {
			continue
		}
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}
		for field := range strings.SplitSeq(rest, ",") {
			if k, v, ok := strings.Cut(field, "="); ok && k == "keys" {
				n, _ := strconv.ParseInt(v, 10, 64)
				counts[idx] = n
			}
		}
	}

	out := make([]driver.DatabaseInfo, 0, maxDatabases)
	for i := range maxDatabases {
		out = append(out, driver.DatabaseInfo{Name: strconv.Itoa(i), SizeBytes: counts[i]})
	}
	return out, nil
}

// CreateDatabase is a no-op: Redis's numbered databases all already exist
// (there is no way to "create" database N that isn't already selectable).
// initialCollection is ignored — it has no Redis equivalent.
func (c *Client) CreateDatabase(ctx context.Context, dbName, initialCollection string) error {
	if _, err := dbIndex(dbName); err != nil {
		return err
	}
	return nil
}

// DropDatabase runs FLUSHDB on the given database index, deleting every
// key in it. There is no more granular "drop" available in Redis — this
// is the literal, complete-erasure operation, not a soft delete.
func (c *Client) DropDatabase(ctx context.Context, dbName string) error {
	idx, err := dbIndex(dbName)
	if err != nil {
		return err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return err
	}
	if err := rc.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("flushdb %d: %w", idx, err)
	}
	return nil
}
