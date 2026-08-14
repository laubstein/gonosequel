package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestCollectionLifecycle(t *testing.T) {
	c, ctx := newTestClient(t, "test_coll_lifecycle")

	if err := c.CreateCollection(ctx, "test_coll_lifecycle", "widgets", driver.CreateCollectionOptions{}); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	colls, err := c.ListCollections(ctx, "test_coll_lifecycle")
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(colls) != 1 || colls[0].Name != "widgets" {
		t.Fatalf("expected [widgets], got %+v", colls)
	}

	if err := c.RenameCollection(ctx, "test_coll_lifecycle", "widgets", "gadgets"); err != nil {
		t.Fatalf("RenameCollection: %v", err)
	}
	colls, err = c.ListCollections(ctx, "test_coll_lifecycle")
	if err != nil {
		t.Fatalf("ListCollections after rename: %v", err)
	}
	if len(colls) != 1 || colls[0].Name != "gadgets" {
		t.Fatalf("expected [gadgets] after rename, got %+v", colls)
	}

	if _, err := c.InsertOne(ctx, "test_coll_lifecycle", "gadgets", bson.M{"n": 1}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}
	stats, err := c.Stats(ctx, "test_coll_lifecycle", "gadgets")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Count != 1 {
		t.Errorf("expected count 1, got %d", stats.Count)
	}

	if err := c.DropCollection(ctx, "test_coll_lifecycle", "gadgets"); err != nil {
		t.Fatalf("DropCollection: %v", err)
	}
	colls, err = c.ListCollections(ctx, "test_coll_lifecycle")
	if err != nil {
		t.Fatalf("ListCollections after drop: %v", err)
	}
	if len(colls) != 0 {
		t.Fatalf("expected no collections after drop, got %+v", colls)
	}
}
