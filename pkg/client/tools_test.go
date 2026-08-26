package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestCollectionsOverview(t *testing.T) {
	c, ctx := newTestClient(t, "test_tools_overview")

	if _, err := c.InsertOne(ctx, "test_tools_overview", "a", bson.M{"x": 1}); err != nil {
		t.Fatalf("InsertOne a: %v", err)
	}
	if _, err := c.InsertOne(ctx, "test_tools_overview", "b", bson.M{"x": 1}); err != nil {
		t.Fatalf("InsertOne b: %v", err)
	}

	overview, err := c.CollectionsOverview(ctx, "test_tools_overview")
	if err != nil {
		t.Fatalf("CollectionsOverview: %v", err)
	}
	if len(overview) != 2 {
		t.Fatalf("expected 2 collections, got %d: %+v", len(overview), overview)
	}

	names := map[string]bool{}
	for _, s := range overview {
		names[s.Name] = true
		if s.Count != 1 {
			t.Errorf("collection %q: expected count 1, got %d", s.Name, s.Count)
		}
	}
	if !names["a"] || !names["b"] {
		t.Errorf("expected collections a and b, got %+v", names)
	}
}

func TestIndexUsage(t *testing.T) {
	c, ctx := newTestClient(t, "test_tools_indexusage")

	if _, err := c.InsertOne(ctx, "test_tools_indexusage", "items", bson.M{"sku": "abc"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}
	if _, err := c.CreateIndex(ctx, "test_tools_indexusage", "items", driver.OrderedDoc{{Key: "sku", Value: 1}}, driver.CreateIndexOptions{}); err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}

	usage, err := c.IndexUsage(ctx, "test_tools_indexusage")
	if err != nil {
		t.Fatalf("IndexUsage: %v", err)
	}

	// _id_ plus the sku index we just created; neither has necessarily
	// been used by a query yet (Ops can legitimately be 0), we're only
	// checking that both show up with a decodable Since timestamp.
	if len(usage) != 2 {
		t.Fatalf("expected 2 index usage entries, got %d: %+v", len(usage), usage)
	}
	names := map[string]bool{}
	for _, u := range usage {
		names[u.Index] = true
		if u.Collection != "items" {
			t.Errorf("expected collection 'items', got %q", u.Collection)
		}
		if u.Since.IsZero() {
			t.Errorf("expected a non-zero Since for index %q", u.Index)
		}
	}
	if !names["_id_"] || !names["sku_1"] {
		t.Errorf("expected _id_ and sku_1 indexes, got %+v", names)
	}
}

func TestCurrentOps(t *testing.T) {
	c, ctx := newTestClient(t, "test_tools_currentops")

	// Not asserting on contents (there's no reliable way to keep a
	// long-running op alive from this test), just that a very high
	// minSecs threshold filters everything out and the call itself
	// succeeds against a live server.
	ops, err := c.CurrentOps(ctx, 3600)
	if err != nil {
		t.Fatalf("CurrentOps: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("expected no operations running for over an hour, got %+v", ops)
	}
}
