package client

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestIndexLifecycle(t *testing.T) {
	c, ctx := newTestClient(t, "test_index_lifecycle")

	if _, err := c.InsertOne(ctx, "test_index_lifecycle", "products", bson.M{"sku": "abc"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	name, err := c.CreateIndex(ctx, "test_index_lifecycle", "products", driver.OrderedDoc{{Key: "sku", Value: 1}}, driver.CreateIndexOptions{Unique: true})
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if name == "" {
		t.Fatal("expected a non-empty index name")
	}

	indexes, err := c.ListIndexes(ctx, "test_index_lifecycle", "products")
	if err != nil {
		t.Fatalf("ListIndexes: %v", err)
	}
	// _id_ is always present, plus the one we just created.
	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes (_id_ + sku), got %+v", indexes)
	}

	if err := c.DropIndex(ctx, "test_index_lifecycle", "products", name); err != nil {
		t.Fatalf("DropIndex: %v", err)
	}
	indexes, err = c.ListIndexes(ctx, "test_index_lifecycle", "products")
	if err != nil {
		t.Fatalf("ListIndexes after drop: %v", err)
	}
	if len(indexes) != 1 {
		t.Fatalf("expected only _id_ index after drop, got %+v", indexes)
	}
}

func TestIndexOptionsRoundTrip(t *testing.T) {
	c, ctx := newTestClient(t, "test_index_options")

	if _, err := c.InsertOne(ctx, "test_index_options", "events", bson.M{"createdAt": time.Now(), "age": 30}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	// MongoDB rejects an index that combines sparse and
	// partialFilterExpression ("cannot mix" error), so this test covers
	// TTL + partial filter together; sparse is covered on its own below.
	ttl := int64(3600)
	opts := driver.CreateIndexOptions{
		ExpireAfterSeconds:      &ttl,
		PartialFilterExpression: driver.Doc{"age": driver.Doc{"$gt": 18}},
	}
	name, err := c.CreateIndex(ctx, "test_index_options", "events", driver.OrderedDoc{{Key: "createdAt", Value: 1}}, opts)
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}

	indexes, err := c.ListIndexes(ctx, "test_index_options", "events")
	if err != nil {
		t.Fatalf("ListIndexes: %v", err)
	}

	var got *driver.IndexInfo
	for i := range indexes {
		if indexes[i].Name == name {
			got = &indexes[i]
		}
	}
	if got == nil {
		t.Fatalf("index %q not found in %+v", name, indexes)
	}
	if got.ExpireAfterSeconds == nil || *got.ExpireAfterSeconds != ttl {
		t.Errorf("ExpireAfterSeconds = %v, want %d", got.ExpireAfterSeconds, ttl)
	}
	if got.PartialFilterExpression == nil {
		t.Errorf("PartialFilterExpression is nil, want the configured filter to round-trip")
	}
}

func TestIndexSparseOption(t *testing.T) {
	c, ctx := newTestClient(t, "test_index_sparse")

	if _, err := c.InsertOne(ctx, "test_index_sparse", "events", bson.M{"maybeField": "x"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	name, err := c.CreateIndex(ctx, "test_index_sparse", "events", driver.OrderedDoc{{Key: "maybeField", Value: 1}}, driver.CreateIndexOptions{Sparse: true})
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}

	indexes, err := c.ListIndexes(ctx, "test_index_sparse", "events")
	if err != nil {
		t.Fatalf("ListIndexes: %v", err)
	}
	var got *driver.IndexInfo
	for i := range indexes {
		if indexes[i].Name == name {
			got = &indexes[i]
		}
	}
	if got == nil || !got.Sparse {
		t.Fatalf("expected Sparse=true, got %+v", got)
	}
}

func TestUpdateIndexTTL(t *testing.T) {
	c, ctx := newTestClient(t, "test_update_index_ttl")

	if _, err := c.InsertOne(ctx, "test_update_index_ttl", "events", bson.M{"createdAt": time.Now()}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	ttl := int64(3600)
	name, err := c.CreateIndex(ctx, "test_update_index_ttl", "events", driver.OrderedDoc{{Key: "createdAt", Value: 1}}, driver.CreateIndexOptions{ExpireAfterSeconds: &ttl})
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}

	if err := c.UpdateIndexTTL(ctx, "test_update_index_ttl", "events", name, 7200); err != nil {
		t.Fatalf("UpdateIndexTTL: %v", err)
	}

	indexes, err := c.ListIndexes(ctx, "test_update_index_ttl", "events")
	if err != nil {
		t.Fatalf("ListIndexes: %v", err)
	}
	var got *driver.IndexInfo
	for i := range indexes {
		if indexes[i].Name == name {
			got = &indexes[i]
		}
	}
	if got == nil || got.ExpireAfterSeconds == nil || *got.ExpireAfterSeconds != 7200 {
		t.Fatalf("expected ExpireAfterSeconds=7200 after update, got %+v", got)
	}
}
