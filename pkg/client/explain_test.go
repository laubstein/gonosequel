package client

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestExplain(t *testing.T) {
	c, ctx := newTestClient(t, "test_explain")

	if _, err := c.InsertOne(ctx, "test_explain", "items", bson.M{"n": int32(1)}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	result, err := c.Explain(ctx, "test_explain", "items", driver.FindOptions{
		Filter: bson.M{"n": int32(1)},
	})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}

	if _, ok := result["queryPlanner"]; !ok {
		t.Errorf("expected queryPlanner in explain output, got keys: %v", mapKeys(result))
	}
	if _, ok := result["executionStats"]; !ok {
		t.Errorf("expected executionStats in explain output, got keys: %v", mapKeys(result))
	}
}

// TestExplainIncludesSortAndProjection pins that Explain describes the
// whole query, not just its filter: a sort with no index behind it adds a
// blocking SORT stage that explaining the filter alone would never reveal.
func TestExplainIncludesSortAndProjection(t *testing.T) {
	c, ctx := newTestClient(t, "test_explain_sort")

	if _, err := c.InsertOne(ctx, "test_explain_sort", "items", bson.M{"n": int32(1), "other": "x"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	result, err := c.Explain(ctx, "test_explain_sort", "items", driver.FindOptions{
		Sort:       driver.OrderedDoc{{Key: "other", Value: int32(1)}},
		Projection: bson.M{"other": int32(0)},
	})
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}

	raw, err := bson.MarshalExtJSON(result, false, false)
	if err != nil {
		t.Fatalf("marshal explain: %v", err)
	}
	if !bytes.Contains(raw, []byte("SORT")) {
		t.Errorf("expected a SORT stage in the plan for an unindexed sort, got: %s", raw)
	}
	if !bytes.Contains(raw, []byte("PROJECTION")) {
		t.Errorf("expected a PROJECTION stage in the plan, got: %s", raw)
	}
}

func mapKeys(m bson.M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
