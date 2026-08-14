// Package export streams query results out of MongoDB in JSON or CSV
// form without materializing the whole result set in memory.
package export

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/client"
)

// JSON writes each document from docs to w as a single JSON array of
// relaxed Extended JSON objects, one document at a time.
func JSON(w *bufio.Writer, docs []bson.M) error {
	if _, err := w.WriteString("["); err != nil {
		return err
	}
	for i, doc := range docs {
		if i > 0 {
			if _, err := w.WriteString(","); err != nil {
				return err
			}
		}
		raw, err := client.ToRelaxedExtJSON(doc)
		if err != nil {
			return fmt.Errorf("marshal document %d: %w", i, err)
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
	}
	_, err := w.WriteString("]")
	return err
}

// CSV writes docs to w as CSV, flattening nested paths into dotted column
// names (e.g. address.city) and deriving the column set from the union of
// fields across all documents, sorted for a stable column order.
func CSV(w *bufio.Writer, docs []bson.M) error {
	columns := csvColumns(docs)

	cw := csv.NewWriter(w)
	if err := cw.Write(columns); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, doc := range docs {
		flat := flatten(doc, "")
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = flat[col]
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}

func csvColumns(docs []bson.M) []string {
	set := map[string]struct{}{}
	for _, doc := range docs {
		for k := range flatten(doc, "") {
			set[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(set))
	for k := range set {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

// flatten converts a nested BSON document into a flat map of dotted paths
// to string representations, suitable for a CSV row.
func flatten(doc bson.M, prefix string) map[string]string {
	out := map[string]string{}
	for k, v := range doc {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch val := v.(type) {
		case bson.M:
			for nk, nv := range flatten(val, path) {
				out[nk] = nv
			}
		default:
			out[path] = stringify(v)
		}
	}
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if raw, err := client.ToRelaxedExtJSON(bson.M{"v": v}); err == nil {
		s := string(raw)
		return strings.TrimSuffix(strings.TrimPrefix(s, `{"v":`), "}")
	}
	return fmt.Sprintf("%v", v)
}
