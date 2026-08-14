package client

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestDocumentCRUD(t *testing.T) {
	c, ctx := newTestClient(t, "test_doc_crud")

	dec128, err := bson.ParseDecimal128("42.5")
	if err != nil {
		t.Fatalf("ParseDecimal128: %v", err)
	}

	doc := bson.M{
		"name":     "Ana",
		"active":   true,
		"score":    int64(9223372036854775800),
		"amount":   dec128,
		"joinedAt": bson.NewDateTimeFromTime(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
	}

	id, err := c.InsertOne(ctx, "test_doc_crud", "users", doc)
	if err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	got, err := c.FindOne(ctx, "test_doc_crud", "users", id)
	if err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	if got["name"] != "Ana" {
		t.Errorf("expected name Ana, got %v", got["name"])
	}
	// The persisted-and-reread value must keep its exact BSON type, not
	// just an equal-looking value.
	if _, ok := got["score"].(int64); !ok {
		t.Errorf("expected score to round-trip as int64, got %T", got["score"])
	}
	if _, ok := got["amount"].(bson.Decimal128); !ok {
		t.Errorf("expected amount to round-trip as Decimal128, got %T", got["amount"])
	}

	got["name"] = "Ana Maria"
	if err := c.ReplaceOne(ctx, "test_doc_crud", "users", id, got); err != nil {
		t.Fatalf("ReplaceOne: %v", err)
	}
	updated, err := c.FindOne(ctx, "test_doc_crud", "users", id)
	if err != nil {
		t.Fatalf("FindOne after replace: %v", err)
	}
	if updated["name"] != "Ana Maria" {
		t.Errorf("expected name Ana Maria after replace, got %v", updated["name"])
	}

	if err := c.DeleteOne(ctx, "test_doc_crud", "users", id); err != nil {
		t.Fatalf("DeleteOne: %v", err)
	}
	_, err = c.FindOne(ctx, "test_doc_crud", "users", id)
	if !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestFindPagination(t *testing.T) {
	c, ctx := newTestClient(t, "test_find_pagination")

	for i := 0; i < 25; i++ {
		if _, err := c.InsertOne(ctx, "test_find_pagination", "items", bson.M{"n": int32(i)}); err != nil {
			t.Fatalf("InsertOne %d: %v", i, err)
		}
	}

	page1, err := c.Find(ctx, "test_find_pagination", "items", driver.FindOptions{
		Sort:  driver.OrderedDoc{{Key: "n", Value: 1}},
		Skip:  0,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Find page1: %v", err)
	}
	if len(page1.Documents) != 10 {
		t.Fatalf("expected 10 docs on page1, got %d", len(page1.Documents))
	}
	if !page1.TotalIsEstimate {
		t.Error("expected TotalIsEstimate=true for an unfiltered query")
	}
	if page1.Documents[0]["n"] != int32(0) {
		t.Errorf("expected first doc n=0, got %v", page1.Documents[0]["n"])
	}

	page3, err := c.Find(ctx, "test_find_pagination", "items", driver.FindOptions{
		Sort:  driver.OrderedDoc{{Key: "n", Value: 1}},
		Skip:  20,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Find page3: %v", err)
	}
	if len(page3.Documents) != 5 {
		t.Fatalf("expected 5 docs on page3 (25 total, skip 20), got %d", len(page3.Documents))
	}

	filtered, err := c.Find(ctx, "test_find_pagination", "items", driver.FindOptions{
		Filter: bson.M{"n": bson.M{"$gte": int32(20)}},
	})
	if err != nil {
		t.Fatalf("Find filtered: %v", err)
	}
	if filtered.TotalIsEstimate {
		t.Error("expected TotalIsEstimate=false for a filtered query")
	}
	if filtered.Total != 5 {
		t.Errorf("expected exact total 5 for filtered query, got %d", filtered.Total)
	}
}
