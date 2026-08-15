package redis

import "testing"

func TestRunCommand(t *testing.T) {
	c, ctx := newTestClient(t, 13)

	if _, err := c.RunCommand(ctx, "13", []string{"SET", "foo", "bar"}); err != nil {
		t.Fatalf("SET: %v", err)
	}

	got, err := c.RunCommand(ctx, "13", []string{"GET", "foo"})
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if got != "bar" {
		t.Errorf("expected \"bar\", got %v (%T)", got, got)
	}
}

func TestRunCommandError(t *testing.T) {
	c, ctx := newTestClient(t, 14)

	if _, err := c.RunCommand(ctx, "14", []string{"SET", "notanumber", "hello"}); err != nil {
		t.Fatalf("SET: %v", err)
	}
	if _, err := c.RunCommand(ctx, "14", []string{"INCR", "notanumber"}); err == nil {
		t.Error("expected an error incrementing a non-numeric string, got nil")
	}
}

func TestRunCommandHGETALL(t *testing.T) {
	c, ctx := newTestClient(t, 15)

	if _, err := c.RunCommand(ctx, "15", []string{"HSET", "h", "a", "1", "b", "2"}); err != nil {
		t.Fatalf("HSET: %v", err)
	}
	got, err := c.RunCommand(ctx, "15", []string{"HGETALL", "h"})
	if err != nil {
		t.Fatalf("HGETALL: %v", err)
	}
	// go-redis's generic Do() decodes a RESP3 map-shaped reply (which
	// HGETALL is) as map[any]any — normalizeCommandResult converts that to
	// map[string]any, both because encoding/json can't marshal
	// map[any]any at all, and so the API's JSON response is actually
	// usable by the frontend.
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected a map[string]any reply, got %T: %+v", got, got)
	}
	if m["a"] != "1" || m["b"] != "2" {
		t.Errorf("expected {a:1 b:2}, got %+v", m)
	}
}
