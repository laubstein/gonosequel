package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestUpdateMany mirrors db.suaColecao.updateMany({activity: null}, {$set:
// {activity: []}}) — set a field on every document where it's currently
// null/missing, leaving documents that already have a non-null value
// untouched.
func TestUpdateMany(t *testing.T) {
	c, ctx := newTestClient(t, "test_update_many")

	if _, err := c.InsertOne(ctx, "test_update_many", "users", bson.M{"_id": "a", "activity": nil}); err != nil {
		t.Fatalf("InsertOne a: %v", err)
	}
	if _, err := c.InsertOne(ctx, "test_update_many", "users", bson.M{"_id": "b"}); err != nil {
		t.Fatalf("InsertOne b: %v", err)
	}
	if _, err := c.InsertOne(ctx, "test_update_many", "users", bson.M{"_id": "c", "activity": bson.A{"login"}}); err != nil {
		t.Fatalf("InsertOne c: %v", err)
	}

	matched, modified, err := c.UpdateMany(
		ctx, "test_update_many", "users",
		bson.M{"activity": nil},
		bson.M{"$set": bson.M{"activity": bson.A{}}},
	)
	if err != nil {
		t.Fatalf("UpdateMany: %v", err)
	}
	if matched != 2 {
		t.Errorf("expected 2 matched (a: explicit null, b: missing field also matches {activity: null}), got %d", matched)
	}
	if modified != 2 {
		t.Errorf("expected 2 modified, got %d", modified)
	}

	a, err := c.FindOne(ctx, "test_update_many", "users", "a")
	if err != nil {
		t.Fatalf("FindOne a: %v", err)
	}
	if arr, ok := a["activity"].([]any); !ok || len(arr) != 0 {
		t.Errorf("expected a.activity to be an empty array, got %#v", a["activity"])
	}

	b, err := c.FindOne(ctx, "test_update_many", "users", "b")
	if err != nil {
		t.Fatalf("FindOne b: %v", err)
	}
	if arr, ok := b["activity"].([]any); !ok || len(arr) != 0 {
		t.Errorf("expected b.activity to be an empty array, got %#v", b["activity"])
	}

	cDoc, err := c.FindOne(ctx, "test_update_many", "users", "c")
	if err != nil {
		t.Fatalf("FindOne c: %v", err)
	}
	if arr, ok := cDoc["activity"].([]any); !ok || len(arr) != 1 || arr[0] != "login" {
		t.Errorf("expected c.activity to remain [\"login\"] (not matched by the filter), got %#v", cDoc["activity"])
	}
}
