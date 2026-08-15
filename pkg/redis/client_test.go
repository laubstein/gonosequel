package redis

import (
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// testURI is the connection string for the shared Redis container used by
// every integration test in this package, mirroring pkg/client's TestMain
// pattern (one container per package run, not per test).
var testURI string

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := tcredis.Run(ctx, "redis:8")
	if err != nil {
		log.Fatalf("start redis container: %v", err)
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

// newTestClient connects to the shared test container. Each test picks a
// dedicated database index (0-15) to avoid interference between parallel
// tests — there's no name-based isolation the way MongoDB's per-test
// database names give pkg/client's tests, since Redis only has numbered
// databases.
func newTestClient(t *testing.T, dbIndex int) (*Client, context.Context) {
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
		_ = c.DropDatabase(ctx, strconv.Itoa(dbIndex))
		_ = c.Close(ctx)
	})
	return c, ctx
}
