package redis

import (
	"errors"
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestCreateCollectionUnsupported(t *testing.T) {
	c, ctx := newTestClient(t, 6)
	err := c.CreateCollection(ctx, "6", "widgets", driver.CreateCollectionOptions{})
	if !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("expected ErrUnsupported, got %v", err)
	}
}

func TestDropAndRenameCollection(t *testing.T) {
	c, ctx := newTestClient(t, 7)

	for _, key := range []string{"widgets:1", "widgets:2", "widgets:3"} {
		if _, err := c.InsertOne(ctx, "7", "widgets", driver.Doc{
			"_id": key, "type": "string", "value": "v",
		}); err != nil {
			t.Fatalf("InsertOne %q: %v", key, err)
		}
	}

	if err := c.RenameCollection(ctx, "7", "widgets", "gadgets"); err != nil {
		t.Fatalf("RenameCollection: %v", err)
	}
	if _, err := c.FindOne(ctx, "7", "gadgets", "gadgets:1"); err != nil {
		t.Fatalf("FindOne after rename: %v", err)
	}
	if _, err := c.FindOne(ctx, "7", "widgets", "widgets:1"); !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("expected the old key to be gone after rename, got %v", err)
	}

	stats, err := c.Stats(ctx, "7", "gadgets")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Count != 3 {
		t.Errorf("expected count 3, got %d", stats.Count)
	}

	if err := c.DropCollection(ctx, "7", "gadgets"); err != nil {
		t.Fatalf("DropCollection: %v", err)
	}
	if _, err := c.Stats(ctx, "7", "gadgets"); !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("expected ErrNotFound stats-ing a dropped collection, got %v", err)
	}
}
