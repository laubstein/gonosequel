package redis

import "testing"

func TestDatabaseLifecycle(t *testing.T) {
	c, ctx := newTestClient(t, 1)

	dbs, err := c.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("ListDatabases: %v", err)
	}
	if len(dbs) != maxDatabases {
		t.Fatalf("expected %d databases listed, got %d", maxDatabases, len(dbs))
	}

	if err := c.CreateDatabase(ctx, "1", ""); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}

	if _, err := c.InsertOne(ctx, "1", "widgets", map[string]any{
		"type": "string", "value": "x",
	}); err != nil {
		t.Fatalf("InsertOne: %v", err)
	}

	if err := c.DropDatabase(ctx, "1"); err != nil {
		t.Fatalf("DropDatabase: %v", err)
	}

	colls, err := c.ListCollections(ctx, "1")
	if err != nil {
		t.Fatalf("ListCollections after DropDatabase: %v", err)
	}
	if len(colls) != 0 {
		t.Errorf("expected no collections after FlushDB, got %+v", colls)
	}
}

func TestDatabaseInvalidIndex(t *testing.T) {
	c, ctx := newTestClient(t, 2)
	if _, err := c.ListCollections(ctx, "not-a-number"); err == nil {
		t.Error("expected an error for a non-numeric database name")
	}
}
