package client

import (
	"testing"
)

func TestDatabaseLifecycle(t *testing.T) {
	c, ctx := newTestClient(t, "test_db_lifecycle")

	if err := c.CreateDatabase(ctx, "test_db_lifecycle", "seed"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}

	dbs, err := c.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("ListDatabases: %v", err)
	}
	found := false
	for _, d := range dbs {
		if d.Name == "test_db_lifecycle" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected test_db_lifecycle to appear in ListDatabases, got %+v", dbs)
	}

	if err := c.DropDatabase(ctx, "test_db_lifecycle"); err != nil {
		t.Fatalf("DropDatabase: %v", err)
	}

	dbs, err = c.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("ListDatabases after drop: %v", err)
	}
	for _, d := range dbs {
		if d.Name == "test_db_lifecycle" {
			t.Errorf("expected test_db_lifecycle to be gone after DropDatabase, still present")
		}
	}
}
