package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Explain runs the given find filter through MongoDB's query planner and
// returns the raw explain output, letting the UI show which plan and
// indexes the server chose without executing the query for real.
func (c *Client) Explain(ctx context.Context, dbName, collName string, filter bson.M) (bson.M, error) {
	if filter == nil {
		filter = bson.M{}
	}

	cmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: collName},
			{Key: "filter", Value: filter},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}

	var result bson.M
	if err := c.mongo.Database(dbName).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return nil, fmt.Errorf("explain %q.%q: %w", dbName, collName, err)
	}
	return result, nil
}
