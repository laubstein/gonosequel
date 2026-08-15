package redis

import (
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestInferSchema(t *testing.T) {
	c, ctx := newTestClient(t, 9)

	if _, err := c.InsertOne(ctx, "9", "people", driver.Doc{
		"_id": "people:1", "type": "hash", "value": map[string]any{"name": "Ana", "age": "30"},
	}); err != nil {
		t.Fatalf("InsertOne hash: %v", err)
	}
	if _, err := c.InsertOne(ctx, "9", "people", driver.Doc{
		"_id": "people:2", "type": "string", "value": "just a string",
	}); err != nil {
		t.Fatalf("InsertOne string: %v", err)
	}

	fields, err := c.InferSchema(ctx, "9", "people", 10)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("expected at least one schema field")
	}

	var typeField *driver.SchemaField
	for i := range fields {
		if fields[i].Path == "(key type)" {
			typeField = &fields[i]
		}
	}
	if typeField == nil {
		t.Fatal("expected a (key type) field")
	}
	counts := map[string]int{}
	for _, ft := range typeField.Types {
		counts[ft.Type] = ft.Count
	}
	if counts["hash"] != 1 || counts["string"] != 1 {
		t.Errorf("expected 1 hash + 1 string, got %+v", counts)
	}
}
