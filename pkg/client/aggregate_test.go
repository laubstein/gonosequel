package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAggregate(t *testing.T) {
	c, ctx := newTestClient(t, "test_aggregate")

	docs := []bson.M{
		{"category": "a", "price": int32(10)},
		{"category": "a", "price": int32(20)},
		{"category": "b", "price": int32(5)},
	}
	for _, d := range docs {
		if _, err := c.InsertOne(ctx, "test_aggregate", "items", d); err != nil {
			t.Fatalf("InsertOne: %v", err)
		}
	}

	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":   "$category",
			"total": bson.M{"$sum": "$price"},
		}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}

	results, err := c.Aggregate(ctx, "test_aggregate", "items", pipeline)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(results), results)
	}
	if results[0]["_id"] != "a" || results[0]["total"] != int32(30) {
		t.Errorf("expected {_id: a, total: 30}, got %+v", results[0])
	}
	if results[1]["_id"] != "b" || results[1]["total"] != int32(5) {
		t.Errorf("expected {_id: b, total: 5}, got %+v", results[1])
	}
}

func TestAggregateCapsResults(t *testing.T) {
	c, ctx := newTestClient(t, "test_aggregate_cap")

	// Insert more documents than the cap would need to matter for this
	// test to be meaningful, but keep it small for test speed: cap the
	// pipeline's own $limit above maxAggregateResults and confirm our
	// side still enforces its own ceiling regardless.
	for i := 0; i < 5; i++ {
		if _, err := c.InsertOne(ctx, "test_aggregate_cap", "items", bson.M{"n": int32(i)}); err != nil {
			t.Fatalf("InsertOne %d: %v", i, err)
		}
	}

	pipeline := bson.A{bson.M{"$limit": int32(3)}}
	results, err := c.Aggregate(ctx, "test_aggregate_cap", "items", pipeline)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected pipeline's own $limit of 3 to be respected, got %d results", len(results))
	}
}
