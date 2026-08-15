package redis

import "testing"

func TestServerStatus(t *testing.T) {
	c, ctx := newTestClient(t, 10)

	status, err := c.ServerStatus(ctx)
	if err != nil {
		t.Fatalf("ServerStatus: %v", err)
	}
	if status.Version == "" {
		t.Error("expected a non-empty version")
	}
	if status.Connections.Current < 1 {
		t.Errorf("expected connections.current >= 1, got %d", status.Connections.Current)
	}
}

func TestCurrentOps(t *testing.T) {
	c, ctx := newTestClient(t, 11)

	// Just confirms the call succeeds against a live server — see
	// tools.go's doc comment on why this is usually empty.
	if _, err := c.CurrentOps(ctx, 3600); err != nil {
		t.Fatalf("CurrentOps: %v", err)
	}
}

func TestCollectionsOverview(t *testing.T) {
	c, ctx := newTestClient(t, 12)

	if _, err := c.InsertOne(ctx, "12", "a", map[string]any{"type": "string", "value": "x"}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	overview, err := c.CollectionsOverview(ctx, "12")
	if err != nil {
		t.Fatalf("CollectionsOverview: %v", err)
	}
	if len(overview) != 1 || overview[0].Name != "a" || overview[0].Count != 1 {
		t.Errorf("expected [{a count=1}], got %+v", overview)
	}
}
