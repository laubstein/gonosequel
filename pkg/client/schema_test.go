package client

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
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

	byPath := map[string]driver.SchemaField{}
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

// TestInferSchemaDescendsIntoNestedDocuments pins the dotted paths that
// make a nested field reachable from autocomplete and the Schema tab.
// Embedded documents decode as bson.D (not bson.M) when the parent is
// decoded into a bson.M, so matching only bson.M meant nested fields were
// invisible and their type was reported as the literal Go type "bson.D".
func TestInferSchemaDescendsIntoNestedDocuments(t *testing.T) {
	c, ctx := newTestClient(t, "test_schema_nested")

	doc := bson.M{
		"nome": "srv-alpha",
		"SO":   bson.M{"nome": "Ubuntu", "versao": "24.04"},
		"discos": bson.A{
			bson.M{"ponto": "/", "tamanho": int32(512)},
		},
	}
	if _, err := c.InsertOne(ctx, "test_schema_nested", "hosts", doc); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	fields, err := c.InferSchema(ctx, "test_schema_nested", "hosts", 0)
	if err != nil {
		t.Fatalf("InferSchema: %v", err)
	}

	byPath := map[string]string{}
	for _, f := range fields {
		if len(f.Types) > 0 {
			byPath[f.Path] = f.Types[0].Type
		}
	}

	for _, want := range []string{"SO", "SO.nome", "SO.versao", "discos.ponto", "discos.tamanho"} {
		if _, ok := byPath[want]; !ok {
			t.Errorf("expected schema to include %q, got paths %v", want, byPath)
		}
	}
	if got := byPath["SO"]; got != "object" {
		t.Errorf("expected SO to be reported as object, got %q", got)
	}
	if got := byPath["SO.nome"]; got != "string" {
		t.Errorf("expected SO.nome to be a string, got %q", got)
	}
}
