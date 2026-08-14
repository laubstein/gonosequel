package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// maxAggregateResults caps how many documents a single aggregate call
// returns to the API layer, regardless of whether the pipeline itself
// includes a $limit stage — a safety net against an unbounded pipeline
// (e.g. one missing $limit) pulling an entire large collection into
// memory and back out as JSON.
const maxAggregateResults = 1000

// Aggregate runs an aggregation pipeline and returns up to
// maxAggregateResults resulting documents.
func (c *Client) Aggregate(ctx context.Context, dbName, collName string, pipeline bson.A) ([]bson.M, error) {
	cur, err := c.collection(dbName, collName).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate %q.%q: %w", dbName, collName, err)
	}
	defer cur.Close(ctx)

	docs := []bson.M{}
	for cur.Next(ctx) && len(docs) < maxAggregateResults {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode aggregate result: %w", err)
		}
		docs = append(docs, doc)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate aggregate results: %w", err)
	}

	return docs, nil
}
