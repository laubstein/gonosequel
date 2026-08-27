package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Explain runs the given find query through MongoDB's query planner and
// returns the raw explain output, letting the UI show which plan and
// indexes the server chose. Uses executionStats verbosity, so the query
// does run for real, not just get planned.
//
// Sort and projection are part of the explained command, not just the
// filter: a sort can select a different index (or force a blocking
// in-memory SORT stage), and a projection is what makes a query covered.
func (c *Client) Explain(ctx context.Context, dbName, collName string, opts driver.FindOptions) (driver.Doc, error) {
	bsonFilter := toBSON(opts.Filter)
	if bsonFilter == nil {
		bsonFilter = bson.M{}
	}

	find := bson.D{
		{Key: "find", Value: collName},
		{Key: "filter", Value: bsonFilter},
	}
	if len(opts.Sort) > 0 {
		find = append(find, bson.E{Key: "sort", Value: toBSOND(opts.Sort)})
	}
	if proj := toBSON(opts.Projection); proj != nil {
		find = append(find, bson.E{Key: "projection", Value: proj})
	}

	cmd := bson.D{
		{Key: "explain", Value: find},
		{Key: "verbosity", Value: "executionStats"},
	}

	var result bson.M
	if err := c.mongo.Database(dbName).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return nil, fmt.Errorf("explain %q.%q: %w", dbName, collName, err)
	}
	return toDoc(result), nil
}
