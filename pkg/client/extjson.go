// Package client wraps the MongoDB driver: connection management and every
// database operation the API layer needs. It has no knowledge of HTTP.
package client

import (
	"encoding/base64"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ToRelaxedExtJSON marshals a BSON document to relaxed Extended JSON, the
// human-readable form served to the frontend for display.
func ToRelaxedExtJSON(doc bson.M) ([]byte, error) {
	raw, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return nil, fmt.Errorf("marshal relaxed extjson: %w", err)
	}
	return raw, nil
}

// ToCanonicalExtJSON marshals a BSON document to canonical Extended JSON,
// the type-preserving form served when a document is opened for editing so
// that a view-edit-save round-trip cannot silently change a value's type
// (e.g. a Long becoming a Double).
func ToCanonicalExtJSON(doc bson.M) ([]byte, error) {
	raw, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical extjson: %w", err)
	}
	return raw, nil
}

// FromExtJSON unmarshals Extended JSON (relaxed or canonical, both are
// accepted) sent by the frontend back into a BSON document.
func FromExtJSON(raw []byte) (bson.M, error) {
	var doc bson.M
	if err := bson.UnmarshalExtJSON(raw, true, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal extjson: %w", err)
	}
	return doc, nil
}

// FromExtJSONArray unmarshals an Extended JSON array — an aggregation
// pipeline, in practice — sent by the frontend into a BSON array.
func FromExtJSONArray(raw []byte) (bson.A, error) {
	var arr bson.A
	if err := bson.UnmarshalExtJSON(raw, true, &arr); err != nil {
		return nil, fmt.Errorf("unmarshal extjson array: %w", err)
	}
	return arr, nil
}

// EncodeDocID encodes a document's _id (which may be any BSON type, not
// just ObjectID) into a URL-safe string suitable for a route path segment.
func EncodeDocID(id any) (string, error) {
	raw, err := bson.MarshalExtJSON(bson.M{"_id": id}, true, false)
	if err != nil {
		return "", fmt.Errorf("marshal id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeDocID reverses EncodeDocID, returning the original _id value.
func DecodeDocID(encoded string) (any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode id: %w", err)
	}
	var doc struct {
		ID any `bson:"_id"`
	}
	if err := bson.UnmarshalExtJSON(raw, true, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal id: %w", err)
	}
	return doc.ID, nil
}
