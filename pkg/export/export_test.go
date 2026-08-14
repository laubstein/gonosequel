package export

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// fakeCodec implements driver.DocCodec using plain encoding/json, since
// these tests only care about generic document flattening/serialization,
// not any backend's specific Extended JSON rules.
type fakeCodec struct{}

func (fakeCodec) MarshalRelaxed(doc driver.Doc) ([]byte, error)      { return json.Marshal(doc) }
func (fakeCodec) MarshalCanonical(doc driver.Doc) ([]byte, error)    { return json.Marshal(doc) }
func (fakeCodec) UnmarshalDoc(raw []byte) (driver.Doc, error)        { return nil, nil }
func (fakeCodec) UnmarshalDocArray(raw []byte) ([]driver.Doc, error) { return nil, nil }
func (fakeCodec) EncodeDocID(id any) (string, error)                 { return "", nil }
func (fakeCodec) DecodeDocID(encoded string) (any, error)            { return nil, nil }

func TestCSVFlattensNestedPaths(t *testing.T) {
	docs := []driver.Doc{
		{"name": "Ana", "address": driver.Doc{"city": "Recife", "zip": "50000"}},
		{"name": "Bruno", "address": driver.Doc{"city": "Natal"}},
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := CSV(w, docs, fakeCodec{}); err != nil {
		t.Fatalf("CSV: %v", err)
	}
	w.Flush()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d lines: %v", len(lines), lines)
	}

	header := lines[0]
	if !strings.Contains(header, "address.city") || !strings.Contains(header, "address.zip") {
		t.Errorf("expected flattened columns address.city and address.zip in header, got %q", header)
	}
	if strings.Contains(header, "address\n") || strings.Contains(header, "address,") {
		// the raw "address" column itself should not appear, only its
		// flattened children
		t.Errorf("expected no unflattened 'address' column, got %q", header)
	}
}

func TestJSONProducesValidArray(t *testing.T) {
	docs := []driver.Doc{
		{"n": int32(1)},
		{"n": int32(2)},
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := JSON(w, docs, fakeCodec{}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	w.Flush()

	out := buf.String()
	if !strings.HasPrefix(out, "[") || !strings.HasSuffix(out, "]") {
		t.Errorf("expected a JSON array, got %q", out)
	}
	if !strings.Contains(out, `"n":1`) || !strings.Contains(out, `"n":2`) {
		t.Errorf("expected plain numbers, got %q", out)
	}
}

func TestJSONEmptyDocsProducesEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := JSON(w, nil, fakeCodec{}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	w.Flush()

	if buf.String() != "[]" {
		t.Errorf("expected [], got %q", buf.String())
	}
}
