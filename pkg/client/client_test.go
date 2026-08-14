package client

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// testURI is the connection string for the shared MongoDB container used
// by every integration test in this package. It is set up once in
// TestMain to avoid a container-per-test startup cost.
var testURI string

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:8")
	if err != nil {
		log.Fatalf("start mongodb container: %v", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get connection string: %v", err)
	}
	testURI = uri

	code := m.Run()

	if err := testcontainers.TerminateContainer(container); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

// newTestClient connects to the shared test container and registers
// cleanup to drop the given database and close the client. Each test
// should use a unique database name so tests can run in parallel.
func newTestClient(t *testing.T, dbName string) (*Client, context.Context) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	ctx := context.Background()
	c, err := Connect(ctx, testURI)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = c.DropDatabase(ctx, dbName)
		_ = c.Close(ctx)
	})
	return c, ctx
}
