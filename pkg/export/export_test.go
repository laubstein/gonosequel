package export

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCSVFlattensNestedPaths(t *testing.T) {
	docs := []bson.M{
		{"name": "Ana", "address": bson.M{"city": "Recife", "zip": "50000"}},
		{"name": "Bruno", "address": bson.M{"city": "Natal"}},
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := CSV(w, docs); err != nil {
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
	docs := []bson.M{
		{"n": int32(1)},
		{"n": int32(2)},
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := JSON(w, docs); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	w.Flush()

	out := buf.String()
	if !strings.HasPrefix(out, "[") || !strings.HasSuffix(out, "]") {
		t.Errorf("expected a JSON array, got %q", out)
	}
	// Relaxed mode renders plain numbers directly (no $numberInt
	// wrapper) — that's what makes it readable for display/export.
	if !strings.Contains(out, `"n":1`) || !strings.Contains(out, `"n":2`) {
		t.Errorf("expected relaxed extjson plain numbers, got %q", out)
	}
}

func TestJSONEmptyDocsProducesEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := JSON(w, nil); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	w.Flush()

	if buf.String() != "[]" {
		t.Errorf("expected [], got %q", buf.String())
	}
}
