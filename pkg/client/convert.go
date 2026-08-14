package client

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// toDoc recursively converts a bson.M — and any bson.M/bson.D/bson.A
// nested within it — into the generic driver.Doc, so callers outside this
// package never see a concrete BSON container type, only driver.Doc,
// map[string]any, and []any. A shallow, top-level-only conversion would
// leave nested subdocuments as bson.M, which a generic type switch (e.g.
// pkg/export's CSV flattening) would silently fail to recognize.
func toDoc(m bson.M) driver.Doc {
	if m == nil {
		return nil
	}
	out := make(driver.Doc, len(m))
	for k, v := range m {
		out[k] = toAny(v)
	}
	return out
}

func toAny(v any) any {
	switch val := v.(type) {
	case bson.M:
		return toDoc(val)
	case bson.D:
		out := make(driver.Doc, len(val))
		for _, e := range val {
			out[e.Key] = toAny(e.Value)
		}
		return out
	case bson.A:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = toAny(item)
		}
		return out
	default:
		return v
	}
}

// toBSON is the inverse of toDoc — converts a generic driver.Doc, produced
// outside this package, back into bson.M for handing to the mongo driver.
func toBSON(d driver.Doc) bson.M {
	if d == nil {
		return nil
	}
	out := make(bson.M, len(d))
	for k, v := range d {
		out[k] = fromAny(v)
	}
	return out
}

func fromAny(v any) any {
	switch val := v.(type) {
	case driver.Doc:
		return toBSON(val)
	case []any:
		out := make(bson.A, len(val))
		for i, item := range val {
			out[i] = fromAny(item)
		}
		return out
	default:
		return v
	}
}

// toBSOND converts a driver.OrderedDoc into bson.D, for index key specs
// and sort documents where field order matters.
func toBSOND(o driver.OrderedDoc) bson.D {
	if o == nil {
		return nil
	}
	out := make(bson.D, len(o))
	for i, e := range o {
		out[i] = bson.E{Key: e.Key, Value: fromAny(e.Value)}
	}
	return out
}

// toOrderedDoc converts a bson.D into a driver.OrderedDoc.
func toOrderedDoc(d bson.D) driver.OrderedDoc {
	if d == nil {
		return nil
	}
	out := make(driver.OrderedDoc, len(d))
	for i, e := range d {
		out[i] = driver.Entry{Key: e.Key, Value: toAny(e.Value)}
	}
	return out
}
