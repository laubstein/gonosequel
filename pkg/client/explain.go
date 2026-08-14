package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Explain runs the given find filter through MongoDB's query planner and
// returns the raw explain output, letting the UI show which plan and
// indexes the server chose. Uses executionStats verbosity, so the query
// does run for real, not just get planned.
func (c *Client) Explain(ctx context.Context, dbName, collName string, filter driver.Doc) (driver.Doc, error) {
	bsonFilter := toBSON(filter)
	if bsonFilter == nil {
		bsonFilter = bson.M{}
	}

	cmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: collName},
			{Key: "filter", Value: bsonFilter},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}

	var result bson.M
	if err := c.mongo.Database(dbName).RunCommand(ctx, cmd).Decode(&result); err != nil {
		return nil, fmt.Errorf("explain %q.%q: %w", dbName, collName, err)
	}
	return toDoc(result), nil
}
