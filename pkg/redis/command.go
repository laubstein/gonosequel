package redis

import (
	"context"
	"fmt"
)

// RunCommand sends args to the server verbatim via Redis's generic command
// dispatch (RESP's "any command" escape hatch) and returns whatever the
// server replies with — a string, int64, []any, nil, or an error, decoded
// by go-redis exactly as it would be for a typed call. There is no
// validation of args here beyond what Redis itself does: this is the
// backend half of a redis-cli-like console, not a curated API.
func (c *Client) RunCommand(ctx context.Context, dbName string, args []string) (any, error) {
	idx, err := dbIndex(dbName)
	if err != nil {
		return nil, err
	}
	rc, err := c.conn(idx)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	result, err := rc.Do(ctx, toAnySlice(args)...).Result()
	if err != nil {
		return nil, err
	}
	return normalizeCommandResult(result), nil
}

// normalizeCommandResult recursively converts values go-redis's generic Do
// can return but encoding/json can't marshal as-is — map[interface{}]any
// (RESP3 decodes a map-shaped reply like HGETALL this way, not as the flat
// []any RESP2 gives) becomes map[string]any, with keys stringified via
// fmt.Sprint. Everything else passes through unchanged.
func normalizeCommandResult(v any) any {
	switch val := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			out[fmt.Sprint(k)] = normalizeCommandResult(item)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = normalizeCommandResult(item)
		}
		return out
	default:
		return v
	}
}
