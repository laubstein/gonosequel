package client

import (
	"encoding/base64"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// MarshalRelaxed implements driver.DocCodec with relaxed Extended JSON, the
// human-readable form served to the frontend for display.
func (c *Client) MarshalRelaxed(doc driver.Doc) ([]byte, error) {
	raw, err := bson.MarshalExtJSON(toBSON(doc), false, false)
	if err != nil {
		return nil, fmt.Errorf("marshal relaxed extjson: %w", err)
	}
	return raw, nil
}

// MarshalCanonical implements driver.DocCodec with canonical Extended
// JSON, the type-preserving form served when a document is opened for
// editing so that a view-edit-save round-trip cannot silently change a
// value's type (e.g. a Long becoming a Double).
func (c *Client) MarshalCanonical(doc driver.Doc) ([]byte, error) {
	raw, err := bson.MarshalExtJSON(toBSON(doc), true, false)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical extjson: %w", err)
	}
	return raw, nil
}

// UnmarshalDoc implements driver.DocCodec, accepting Extended JSON
// (relaxed or canonical, both are accepted) sent by the frontend.
func (c *Client) UnmarshalDoc(raw []byte) (driver.Doc, error) {
	var doc bson.M
	if err := bson.UnmarshalExtJSON(raw, true, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal extjson: %w", err)
	}
	return toDoc(doc), nil
}

// UnmarshalDocArray implements driver.DocCodec, accepting an Extended JSON
// array — an aggregation pipeline, in practice — sent by the frontend.
func (c *Client) UnmarshalDocArray(raw []byte) ([]driver.Doc, error) {
	var arr bson.A
	if err := bson.UnmarshalExtJSON(raw, true, &arr); err != nil {
		return nil, fmt.Errorf("unmarshal extjson array: %w", err)
	}
	out := make([]driver.Doc, len(arr))
	for i, item := range arr {
		// bson.UnmarshalExtJSON decodes an embedded document inside a
		// bson.A as bson.D (ordered), not bson.M — toAny handles both.
		doc, ok := toAny(item).(driver.Doc)
		if !ok {
			return nil, fmt.Errorf("unmarshal extjson array: element %d is not a document", i)
		}
		out[i] = doc
	}
	return out, nil
}

// EncodeDocID implements driver.DocCodec, encoding a document's _id
// (which may be any BSON type, not just ObjectID) into a URL-safe string
// suitable for a route path segment.
func (c *Client) EncodeDocID(id any) (string, error) {
	raw, err := bson.MarshalExtJSON(bson.M{"_id": id}, true, false)
	if err != nil {
		return "", fmt.Errorf("marshal id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeDocID implements driver.DocCodec, reversing EncodeDocID.
func (c *Client) DecodeDocID(encoded string) (any, error) {
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
