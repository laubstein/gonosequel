package redis

import (
	"errors"
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// TestUnsupportedCapabilitiesReturnErrUnsupported confirms every method
// Redis genuinely can't support fails with driver.ErrUnsupported (so
// pkg/api's errorHandler maps it to 501), not a generic error or a panic.
func TestUnsupportedCapabilitiesReturnErrUnsupported(t *testing.T) {
	c, ctx := newTestClient(t, 8)

	if _, err := c.CreateIndex(ctx, "8", "widgets", driver.OrderedDoc{{Key: "f", Value: 1}}, driver.CreateIndexOptions{}); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("CreateIndex: expected ErrUnsupported, got %v", err)
	}
	if err := c.DropIndex(ctx, "8", "widgets", "some_index"); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("DropIndex: expected ErrUnsupported, got %v", err)
	}
	if err := c.UpdateIndexTTL(ctx, "8", "widgets", "some_index", 60); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("UpdateIndexTTL: expected ErrUnsupported, got %v", err)
	}
	if _, err := c.Explain(ctx, "8", "widgets", driver.FindOptions{}); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("Explain: expected ErrUnsupported, got %v", err)
	}
	if _, err := c.Aggregate(ctx, "8", "widgets", nil); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("Aggregate: expected ErrUnsupported, got %v", err)
	}
	if _, _, err := c.UpdateMany(ctx, "8", "widgets", driver.Doc{}, driver.Doc{}); !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("UpdateMany: expected ErrUnsupported, got %v", err)
	}

	indexes, err := c.ListIndexes(ctx, "8", "widgets")
	if err != nil {
		t.Errorf("ListIndexes: expected nil error (empty list is the honest answer), got %v", err)
	}
	if len(indexes) != 0 {
		t.Errorf("expected no indexes, got %+v", indexes)
	}

	usage, err := c.IndexUsage(ctx, "8")
	if err != nil {
		t.Errorf("IndexUsage: expected nil error, got %v", err)
	}
	if len(usage) != 0 {
		t.Errorf("expected no index usage stats, got %+v", usage)
	}
}

func TestCapabilitiesListMatchesWhatWorks(t *testing.T) {
	c := &Client{}
	caps := c.Capabilities()

	want := map[string]bool{driver.CapFind: true, driver.CapSchema: true, driver.CapTools: true, driver.CapCommand: true}
	if len(caps) != len(want) {
		t.Fatalf("expected %d capabilities, got %+v", len(want), caps)
	}
	for _, capName := range caps {
		if !want[capName] {
			t.Errorf("unexpected capability %q", capName)
		}
	}
	for _, notWant := range []string{driver.CapAggregate, driver.CapExplain, driver.CapIndexes, driver.CapUpdateMany} {
		for _, capName := range caps {
			if capName == notWant {
				t.Errorf("capability %q should not be reported as supported", notWant)
			}
		}
	}
}
