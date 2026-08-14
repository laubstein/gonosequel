package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestExplain(t *testing.T) {
	c, ctx := newTestClient(t, "test_explain")

	if _, err := c.InsertOne(ctx, "test_explain", "items", bson.M{"n": int32(1)}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	result, err := c.Explain(ctx, "test_explain", "items", bson.M{"n": int32(1)})
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

func mapKeys(m bson.M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
