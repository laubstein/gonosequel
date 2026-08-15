package redis

import (
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestDocIDRoundTrip(t *testing.T) {
	codec := &Client{}

	cases := []string{"widgets:1", "with spaces", "with:colons:too", ""}
	for _, key := range cases {
		encoded, err := codec.EncodeDocID(key)
		if err != nil {
			t.Fatalf("EncodeDocID(%q): %v", key, err)
		}
		decoded, err := codec.DecodeDocID(encoded)
		if err != nil {
			t.Fatalf("DecodeDocID(%q): %v", encoded, err)
		}
		if decoded != key {
			t.Errorf("round trip mismatch: original=%q decoded=%v", key, decoded)
		}
	}
}

func TestMarshalRelaxedAndCanonicalAreIdentical(t *testing.T) {
	codec := &Client{}
	doc := driver.Doc{"type": "string", "value": "hello"}

	relaxed, err := codec.MarshalRelaxed(doc)
	if err != nil {
		t.Fatalf("MarshalRelaxed: %v", err)
	}
	canonical, err := codec.MarshalCanonical(doc)
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	if string(relaxed) != string(canonical) {
		t.Errorf("expected relaxed and canonical to be identical for Redis, got %s vs %s", relaxed, canonical)
	}
}

func TestUnmarshalDoc(t *testing.T) {
	codec := &Client{}
	doc, err := codec.UnmarshalDoc([]byte(`{"type":"string","value":"hello"}`))
	if err != nil {
		t.Fatalf("UnmarshalDoc: %v", err)
	}
	if doc["type"] != "string" || doc["value"] != "hello" {
		t.Errorf("unexpected doc: %+v", doc)
	}
}
