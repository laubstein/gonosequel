package client

import "testing"

func TestServerStatus(t *testing.T) {
	c, ctx := newTestClient(t, "test_server_status")

	status, err := c.ServerStatus(ctx)
	if err != nil {
		t.Fatalf("ServerStatus: %v", err)
	}

	if status.Version == "" {
		t.Error("expected a non-empty version")
	}
	if status.Host == "" {
		t.Error("expected a non-empty host")
	}
	// This test's own connection counts as at least one current
	// connection, so this should never be zero against a live server.
	if status.Connections.Current < 1 {
		t.Errorf("expected connections.current >= 1, got %d", status.Connections.Current)
	}
	if status.UptimeSecs < 0 {
		t.Errorf("expected non-negative uptime, got %d", status.UptimeSecs)
	}
}
