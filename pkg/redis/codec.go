package redis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// MarshalRelaxed and MarshalCanonical are identical for Redis: unlike
// MongoDB there are no surrogate extended types (ObjectId, Decimal128,
// ...) to preserve across a view-edit-save round trip — Redis values are
// just strings, and JSON already round-trips those exactly.
func (c *Client) MarshalRelaxed(doc driver.Doc) ([]byte, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	return raw, nil
}

// MarshalCanonical is the same as MarshalRelaxed — see its doc comment.
func (c *Client) MarshalCanonical(doc driver.Doc) ([]byte, error) {
	return c.MarshalRelaxed(doc)
}

// UnmarshalDoc parses a plain JSON document sent by the frontend.
func (c *Client) UnmarshalDoc(raw []byte) (driver.Doc, error) {
	var doc driver.Doc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return doc, nil
}

// UnmarshalDocArray parses a plain JSON array of documents.
func (c *Client) UnmarshalDocArray(raw []byte) ([]driver.Doc, error) {
	var arr []driver.Doc
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("unmarshal json array: %w", err)
	}
	return arr, nil
}

// docIDEnvelope mirrors the shape web/src/api/extjson.ts's encodeDocId
// builds client-side (base64url of `{"_id": <value>}`) — the frontend
// encodes ids itself, for every backend, without a round-trip to ask the
// server how, so every DocCodec implementation has to agree on this exact
// envelope even though Redis itself has no notion of "_id" wrapping.
type docIDEnvelope struct {
	ID string `json:"_id"`
}

// EncodeDocID wraps the raw Redis key in the same {"_id": ...} envelope
// MongoDB's codec uses, base64url-encoded — see docIDEnvelope.
func (c *Client) EncodeDocID(id any) (string, error) {
	key, ok := id.(string)
	if !ok {
		return "", fmt.Errorf("redis document id must be a string key, got %T", id)
	}
	raw, err := json.Marshal(docIDEnvelope{ID: key})
	if err != nil {
		return "", fmt.Errorf("marshal id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeDocID reverses EncodeDocID.
func (c *Client) DecodeDocID(encoded string) (any, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode id: %w", err)
	}
	var env docIDEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("unmarshal id: %w", err)
	}
	return env.ID, nil
}
