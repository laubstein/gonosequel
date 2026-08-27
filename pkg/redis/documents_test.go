package redis

import (
	"errors"
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestDocumentCRUDPerType(t *testing.T) {
	c, ctx := newTestClient(t, 3)

	cases := []struct {
		name  string
		typ   string
		value any
	}{
		{"string", "string", "hello"},
		{"hash", "hash", map[string]any{"a": "1", "b": "2"}},
		{"list", "list", []any{"x", "y", "z"}},
		{"set", "set", []any{"m1", "m2"}},
		{"zset", "zset", []any{
			map[string]any{"member": "alice", "score": float64(1)},
			map[string]any{"member": "bob", "score": float64(2)},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := driver.Doc{"_id": "widgets:" + tc.name, "type": tc.typ, "value": tc.value}
			id, err := c.InsertOne(ctx, "3", "widgets", doc)
			if err != nil {
				t.Fatalf("InsertOne: %v", err)
			}

			got, err := c.FindOne(ctx, "3", "widgets", id)
			if err != nil {
				t.Fatalf("FindOne: %v", err)
			}
			if got["type"] != tc.typ {
				t.Errorf("expected type %q, got %v", tc.typ, got["type"])
			}

			if err := c.DeleteOne(ctx, "3", "widgets", id); err != nil {
				t.Fatalf("DeleteOne: %v", err)
			}
			if _, err := c.FindOne(ctx, "3", "widgets", id); !errors.Is(err, driver.ErrNotFound) {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
		})
	}
}

func TestReplaceOneRequiresExisting(t *testing.T) {
	c, ctx := newTestClient(t, 4)

	err := c.ReplaceOne(ctx, "4", "widgets", "widgets:missing", driver.Doc{"type": "string", "value": "x"})
	if !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("expected ErrNotFound replacing a missing key, got %v", err)
	}
}

func TestFindPaginationAndCollectionGrouping(t *testing.T) {
	c, ctx := newTestClient(t, 5)

	for i := range 25 {
		key := "items:" + string(rune('a'+i))
		if _, err := c.InsertOne(ctx, "5", "items", driver.Doc{
			"_id": key, "type": "string", "value": "v",
		}); err != nil {
			t.Fatalf("InsertOne %d: %v", i, err)
		}
	}
	// A key in a different collection shouldn't show up in "items".
	if _, err := c.InsertOne(ctx, "5", "other", driver.Doc{
		"_id": "other:1", "type": "string", "value": "v",
	}); err != nil {
		t.Fatalf("InsertOne other: %v", err)
	}

	page1, err := c.Find(ctx, "5", "items", driver.FindOptions{Skip: 0, Limit: 10})
	if err != nil {
		t.Fatalf("Find page1: %v", err)
	}
	if page1.Total != 25 {
		t.Errorf("expected total 25, got %d", page1.Total)
	}
	if len(page1.Documents) != 10 {
		t.Errorf("expected 10 docs on page1, got %d", len(page1.Documents))
	}

	page3, err := c.Find(ctx, "5", "items", driver.FindOptions{Skip: 20, Limit: 10})
	if err != nil {
		t.Fatalf("Find page3: %v", err)
	}
	if len(page3.Documents) != 5 {
		t.Errorf("expected 5 docs on page3, got %d", len(page3.Documents))
	}

	colls, err := c.ListCollections(ctx, "5")
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	names := map[string]bool{}
	for _, coll := range colls {
		names[coll.Name] = true
	}
	if !names["items"] || !names["other"] {
		t.Errorf("expected items and other collections, got %+v", colls)
	}
}

// TestKeyTTLRoundTrip covers a feature that used to be half-implemented:
// readKeyDoc reported a key's ttl and the editor displayed it, but no
// write path ever read the field back, so a TTL could not be set at all.
// It also pins the subtler half — every write recreates the key, dropping
// its expiry, so a save that omitted ttl silently made the key permanent.
func TestKeyTTLRoundTrip(t *testing.T) {
	c, ctx := newTestClient(t, 4)

	const key = "widgets:expiring"
	doc := driver.Doc{"_id": key, "type": "string", "value": "v", "ttl": 600}
	if _, err := c.InsertOne(ctx, "0", "widgets", doc); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	got, err := c.FindOne(ctx, "0", "widgets", key)
	if err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	ttl, ok := got["ttl"].(int64)
	if !ok || ttl <= 0 || ttl > 600 {
		t.Fatalf("expected a positive ttl no greater than 600, got %v (%T)", got["ttl"], got["ttl"])
	}

	// ttl <= 0 clears the expiry rather than expiring the key immediately.
	if err := c.ReplaceOne(ctx, "0", "widgets", key, driver.Doc{
		"type": "string", "value": "v2", "ttl": 0,
	}); err != nil {
		t.Fatalf("ReplaceOne: %v", err)
	}
	got, err = c.FindOne(ctx, "0", "widgets", key)
	if err != nil {
		t.Fatalf("FindOne after clearing ttl: %v", err)
	}
	if got["ttl"] != int64(-1) {
		t.Errorf("expected ttl -1 (no expiry) after clearing, got %v", got["ttl"])
	}
	if got["value"] != "v2" {
		t.Errorf("expected the value to be replaced too, got %v", got["value"])
	}
}
