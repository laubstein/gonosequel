package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestInferSchema(t *testing.T) {
	c, ctx := newTestClient(t, "test_infer_schema")

	docs := []bson.M{
		{"name": "Ana", "age": int32(30)},
		{"name": "Bruno", "age": int32(25)},
		{"name": "Carla", "age": "unknown"}, // mixed type on purpose
	}
	for _, d := range docs {
		if _, err := c.InsertOne(ctx, "test_infer_schema", "people", d); err != nil {
			t.Fatalf("InsertOne: %v", err)
		}
	}

	fields, err := c.InferSchema(ctx, "test_infer_schema", "people", 10)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}

	byPath := map[string]SchemaField{}
	for _, f := range fields {
		byPath[f.Path] = f
	}

	nameField, ok := byPath["name"]
	if !ok {
		t.Fatal("expected a 'name' field in inferred schema")
	}
	if len(nameField.Types) != 1 || nameField.Types[0].Type != "string" || nameField.Types[0].Count != 3 {
		t.Errorf("expected name: string x3, got %+v", nameField.Types)
	}

	ageField, ok := byPath["age"]
	if !ok {
		t.Fatal("expected an 'age' field in inferred schema")
	}
	if len(ageField.Types) != 2 {
		t.Errorf("expected age to have 2 distinct types (int + string), got %+v", ageField.Types)
	}
}
