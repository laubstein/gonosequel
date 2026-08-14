package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// FieldType describes one observed BSON type at a field path, with how
// often it was seen in the sample.
type FieldType struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// SchemaField summarizes a single field path across the sampled documents.
type SchemaField struct {
	Path  string      `json:"path"`
	Types []FieldType `json:"types"`
}

// DefaultSchemaSampleSize is the number of documents sampled by
// InferSchema when the caller does not specify a size.
const DefaultSchemaSampleSize = 100

// InferSchema samples up to sampleSize documents from a collection and
// aggregates BSON types observed at each field path. There is no
// declared schema in MongoDB, so this drives autocomplete and the
// frontend's Schema tab instead of an authoritative source of truth.
func (c *Client) InferSchema(ctx context.Context, dbName, collName string, sampleSize int64) ([]SchemaField, error) {
	if sampleSize <= 0 {
		sampleSize = DefaultSchemaSampleSize
	}

	pipeline := bson.A{
		bson.M{"$sample": bson.M{"size": sampleSize}},
	}
	cur, err := c.collection(dbName, collName).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("sample %q.%q: %w", dbName, collName, err)
	}
	defer cur.Close(ctx)

	counts := map[string]map[string]int{}
	order := []string{}

	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode sampled document: %w", err)
		}
		walkFields(doc, "", func(path, bsonType string) {
			if _, ok := counts[path]; !ok {
				counts[path] = map[string]int{}
				order = append(order, path)
			}
			counts[path][bsonType]++
		})
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate sampled documents: %w", err)
	}

	fields := make([]SchemaField, 0, len(order))
	for _, path := range order {
		types := make([]FieldType, 0, len(counts[path]))
		for t, n := range counts[path] {
			types = append(types, FieldType{Type: t, Count: n})
		}
		fields = append(fields, SchemaField{Path: path, Types: types})
	}
	return fields, nil
}

// walkFields recursively visits every field path in doc, calling visit
// with the dotted path and BSON type name for each leaf and nested
// document.
func walkFields(doc bson.M, prefix string, visit func(path, bsonType string)) {
	for key, value := range doc {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		visit(path, bsonTypeName(value))

		if nested, ok := value.(bson.M); ok {
			walkFields(nested, path, visit)
		}
	}
}

// bsonTypeName returns a short, stable name for the Go type produced by
// the driver's default BSON decoding for a value.
func bsonTypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bson.ObjectID:
		return "objectId"
	case string:
		return "string"
	case int32:
		return "int"
	case int64:
		return "long"
	case float64:
		return "double"
	case bool:
		return "bool"
	case bson.DateTime:
		return "date"
	case bson.Decimal128:
		return "decimal"
	case bson.Binary:
		return "binary"
	case bson.A:
		return "array"
	case bson.M:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}
