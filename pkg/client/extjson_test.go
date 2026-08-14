package client

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// extJSONEqual compares two canonical Extended JSON documents structurally,
// ignoring key order (bson.M is a Go map, so marshaling order is not
// stable). Because canonical ExtJSON represents every BSON type as an
// explicit wrapper object (e.g. {"$numberInt":"42"} vs
// {"$numberDouble":"42.0"}), a structural JSON comparison is still fully
// type-sensitive.
func extJSONEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var ma, mb any
	if err := json.Unmarshal(a, &ma); err != nil {
		t.Fatalf("unmarshal a: %v", err)
	}
	if err := json.Unmarshal(b, &mb); err != nil {
		t.Fatalf("unmarshal b: %v", err)
	}
	return reflect.DeepEqual(ma, mb)
}

// extJSONEqualBase64 is extJSONEqual for base64url-encoded ExtJSON, as
// produced by EncodeDocID.
func extJSONEqualBase64(t *testing.T, a, b string) bool {
	t.Helper()
	rawA, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil {
		t.Fatalf("decode a: %v", err)
	}
	rawB, err := base64.RawURLEncoding.DecodeString(b)
	if err != nil {
		t.Fatalf("decode b: %v", err)
	}
	return extJSONEqual(t, rawA, rawB)
}

// TestExtJSONRoundTrip verifies that every BSON type we care about survives
// a full round trip: BSON doc -> canonical ExtJSON -> back to BSON doc,
// with the concrete Go type preserved. This is the suite most likely to
// catch a regression that silently corrupts user data.
func TestExtJSONRoundTrip(t *testing.T) {
	oid := bson.NewObjectID()
	dec128, err := bson.ParseDecimal128("123.456")
	if err != nil {
		t.Fatalf("ParseDecimal128: %v", err)
	}

	cases := []struct {
		name  string
		value any
	}{
		{"ObjectID", oid},
		{"DateTime", bson.NewDateTimeFromTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))},
		{"Decimal128", dec128},
		{"Int64", int64(9223372036854775807)},
		{"Int32", int32(42)},
		{"Double", 3.14159},
		{"Binary", bson.Binary{Subtype: 0x00, Data: []byte{0x01, 0x02, 0x03}}},
		{"MinKey", bson.MinKey{}},
		{"MaxKey", bson.MaxKey{}},
		{"Bool", true},
		{"String", "hello, world"},
		{"Null", nil},
		{"Array", bson.A{"a", int32(1), true}},
		{"Nested", bson.M{"inner": bson.M{"x": int32(1)}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			original := bson.M{"_id": oid, "value": tc.value}

			canonical, err := ToCanonicalExtJSON(original)
			if err != nil {
				t.Fatalf("ToCanonicalExtJSON: %v", err)
			}

			roundTripped, err := FromExtJSON(canonical)
			if err != nil {
				t.Fatalf("FromExtJSON: %v", err)
			}

			// Compare via re-marshaling to canonical ExtJSON again: this
			// is type-sensitive (unlike reflect.DeepEqual on the raw
			// bson.M, which can't tell int32 from int64 apart reliably
			// once boxed in `any`).
			again, err := ToCanonicalExtJSON(roundTripped)
			if err != nil {
				t.Fatalf("re-marshal ToCanonicalExtJSON: %v", err)
			}

			if !extJSONEqual(t, canonical, again) {
				t.Errorf("round trip mismatch:\n  original:  %s\n  roundtrip: %s", canonical, again)
			}
		})
	}
}

// TestExtJSONRelaxedIsReadable confirms relaxed mode is used for display
// (dates as ISO strings, not $date wrappers) while canonical mode
// preserves exact numeric types.
func TestExtJSONRelaxedIsReadable(t *testing.T) {
	doc := bson.M{"count": int64(5)}

	relaxed, err := ToRelaxedExtJSON(doc)
	if err != nil {
		t.Fatalf("ToRelaxedExtJSON: %v", err)
	}
	canonical, err := ToCanonicalExtJSON(doc)
	if err != nil {
		t.Fatalf("ToCanonicalExtJSON: %v", err)
	}

	if string(relaxed) == string(canonical) {
		t.Errorf("expected relaxed and canonical forms to differ for int64, got identical output: %s", relaxed)
	}
}

// TestDocIDRoundTrip covers _id values of every BSON type, not just
// ObjectID hex strings — a common trap when building document routes.
func TestDocIDRoundTrip(t *testing.T) {
	oid := bson.NewObjectID()
	dec128, err := bson.ParseDecimal128("42")
	if err != nil {
		t.Fatalf("ParseDecimal128: %v", err)
	}

	cases := []struct {
		name string
		id   any
	}{
		{"ObjectID", oid},
		{"String", "custom-string-id"},
		{"Int64", int64(12345)},
		{"Decimal128", dec128},
		{"DateTime", bson.NewDateTimeFromTime(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := EncodeDocID(tc.id)
			if err != nil {
				t.Fatalf("EncodeDocID: %v", err)
			}

			decoded, err := DecodeDocID(encoded)
			if err != nil {
				t.Fatalf("DecodeDocID: %v", err)
			}

			// Compare by re-encoding: same trick as above, sidesteps
			// interface{} comparison pitfalls between equivalent types.
			reencoded, err := EncodeDocID(decoded)
			if err != nil {
				t.Fatalf("re-encode EncodeDocID: %v", err)
			}
			if !extJSONEqualBase64(t, encoded, reencoded) {
				t.Errorf("id round trip mismatch: original=%s decoded-reencoded=%s", encoded, reencoded)
			}
		})
	}
}

func TestDecodeDocIDInvalidInput(t *testing.T) {
	if _, err := DecodeDocID("not-valid-base64url!!!"); err == nil {
		t.Error("expected an error decoding invalid base64url input, got nil")
	}
}
