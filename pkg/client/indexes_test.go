package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestIndexLifecycle(t *testing.T) {
	c, ctx := newTestClient(t, "test_index_lifecycle")

	if _, err := c.InsertOne(ctx, "test_index_lifecycle", "products", bson.M{"sku": "abc"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	name, err := c.CreateIndex(ctx, "test_index_lifecycle", "products", driver.OrderedDoc{{Key: "sku", Value: 1}}, true)
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
