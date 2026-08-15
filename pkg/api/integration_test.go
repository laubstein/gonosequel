package api

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/laubstein/gonosequel/pkg/client"
	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/redis"
	"github.com/laubstein/gonosequel/pkg/session"
)

var testMongoURI string
var testRedisURI string

func TestMain(m *testing.M) {
	// go test runs both server_test.go's unit tests and this file's
	// integration tests in one binary; only spin up Docker when we're not
	// in -short mode.
	code := runTestMain(m)
	os.Exit(code)
}

func runTestMain(m *testing.M) int {
	flag.Parse()
	if testing.Short() {
		return m.Run()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:8")
	if err != nil {
		log.Fatalf("start mongodb container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}()

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get connection string: %v", err)
	}
	testMongoURI = uri

	redisContainer, err := tcredis.Run(ctx, "redis:8")
	if err != nil {
		log.Fatalf("start redis container: %v", err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(redisContainer); err != nil {
			log.Printf("terminate redis container: %v", err)
		}
	}()

	redisURI, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get redis connection string: %v", err)
	}
	testRedisURI = redisURI

	return m.Run()
}

// newTestApp builds a fiber app backed by a fresh connection to the shared
// test container, registered as the default session.
func newTestApp(t *testing.T, readonly bool) *fiber.App {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	ctx := context.Background()
	cl, err := client.Connect(ctx, testMongoURI)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cl.Close(context.Background()) })

	registry := session.NewRegistry()
	registry.Put(session.DefaultID, cl, session.Info{ID: session.DefaultID, Name: "test", Readonly: readonly})

	return New(Config{Registry: registry, Readonly: readonly})
}

// newRedisTestApp is newTestApp's Redis equivalent, for the handlers that
// only Redis actually supports (RunCommand).
func newRedisTestApp(t *testing.T, readonly bool) *fiber.App {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	ctx := context.Background()
	cl, err := redis.Connect(ctx, testRedisURI)
	if err != nil {
		t.Fatalf("redis.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cl.Close(context.Background()) })

	registry := session.NewRegistry()
	registry.Put(session.DefaultID, cl, session.Info{ID: session.DefaultID, Name: "test", Readonly: readonly})

	return New(Config{Registry: registry, Driver: "redis", Readonly: readonly})
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) (*http.Response, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var decoded map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			t.Fatalf("decode response body %q: %v", respBody, err)
		}
	}
	return resp, decoded
}

func TestAPIDatabaseAndCollectionLifecycle(t *testing.T) {
	app := newTestApp(t, false)

	resp, body := doJSON(t, app, http.MethodPost, "/api/databases", map[string]string{
		"name":              "api_test_db",
		"initialCollection": "seed",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create database: status=%d body=%v", resp.StatusCode, body)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	listResp, err := app.Test(listReq)
	if err != nil {
		t.Fatalf("list databases: %v", err)
	}
	listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list databases: status=%d", listResp.StatusCode)
	}

	resp, body = doJSON(t, app, http.MethodPost, "/api/databases/api_test_db/collections", map[string]string{
		"name": "widgets",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	resp, _ = doJSON(t, app, http.MethodDelete, "/api/databases/api_test_db", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drop database: status=%d", resp.StatusCode)
	}
}

func TestAPIDocumentCRUDPreservesExtJSONTypes(t *testing.T) {
	app := newTestApp(t, false)

	if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/api_test_docs/collections", map[string]string{"name": "items"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	insertBody := []byte(`{"name":"Ana","count":{"$numberLong":"9007199254740993"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_docs/collections/items/documents", bytes.NewReader(insertBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}
	insertRespBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("insert document: status=%d body=%s", resp.StatusCode, insertRespBody)
	}
	var insertResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(insertRespBody, &insertResp); err != nil {
		t.Fatalf("decode insert response: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/databases/api_test_docs/collections/items/documents/"+insertResp.ID, nil)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	defer getResp.Body.Close()
	getBody, _ := io.ReadAll(getResp.Body)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get document: status=%d body=%s", getResp.StatusCode, getBody)
	}
	// The canonical form must keep the exact numeric type: $numberLong,
	// not silently downgraded to $numberInt or a bare JSON number.
	if !bytes.Contains(getBody, []byte(`"$numberLong":"9007199254740993"`)) {
		t.Errorf("expected canonical $numberLong to survive the round trip, got: %s", getBody)
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/databases/api_test_docs/collections/items/documents/"+insertResp.ID, nil)
	delResp, err := app.Test(delReq)
	if err != nil {
		t.Fatalf("delete document: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete document: status=%d", delResp.StatusCode)
	}
}

func TestAPIReadonlyRejectsScopedWrites(t *testing.T) {
	app := newTestApp(t, true)

	req := httptest.NewRequest(http.MethodPost, "/api/databases", bytes.NewReader([]byte(`{"name":"nope"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestAPIUnknownSessionRejected(t *testing.T) {
	app := newTestApp(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	req.Header.Set(sessionIDHeader, "does-not-exist")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

// TestAPISessionsModeConnectDispatchesByDriver covers the bug this test
// would have caught: /api/connect used to call pkg/client.Connect
// unconditionally, so --sessions mode could only ever open MongoDB
// connections regardless of what driver the request asked for. It now
// goes through Config.Connect, the same dispatch function main.go uses
// for the startup connection.
func TestAPISessionsModeConnectDispatchesByDriver(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	registry := session.NewRegistry()
	app := New(Config{
		Registry: registry,
		Sessions: true,
		Connect: func(ctx context.Context, driverName, uri string) (driver.Driver, error) {
			if driverName != "mongodb" {
				return nil, fmt.Errorf("unexpected driver %q", driverName)
			}
			return client.Connect(ctx, uri)
		},
	})

	resp, body := doJSON(t, app, http.MethodPost, "/api/connect", map[string]string{
		"url":    testMongoURI,
		"driver": "mongodb",
		"name":   "test-session",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect: status=%d body=%v", resp.StatusCode, body)
	}
	sessionID, _ := body["sessionId"].(string)
	if sessionID == "" {
		t.Fatalf("expected a sessionId in response, got %v", body)
	}
	t.Cleanup(func() {
		if cl, err := registry.Get(sessionID); err == nil {
			_ = cl.Close(context.Background())
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	req.Header.Set(sessionIDHeader, sessionID)
	connResp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test /api/connection: %v", err)
	}
	defer connResp.Body.Close()
	if connResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/connection: status = %d, want 200", connResp.StatusCode)
	}
}

// mongoConnectApp is a --sessions mode app whose Connect dispatches only
// to MongoDB, for the per-session readonly tests below — they only care
// about session.Info.Readonly, not driver dispatch.
func mongoConnectApp(t *testing.T) (*fiber.App, *session.Registry) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	registry := session.NewRegistry()
	app := New(Config{
		Registry: registry,
		Sessions: true,
		Connect: func(ctx context.Context, driverName, uri string) (driver.Driver, error) {
			return client.Connect(ctx, uri)
		},
	})
	return app, registry
}

// TestAPIPerSessionReadonlyBlocksWrites covers opting a single --sessions
// connection into read-only from the connect form, independent of the
// server-wide --readonly flag (which is off here) — see
// ConnectionModal.tsx's readonly checkbox and session.Info.Readonly.
func TestAPIPerSessionReadonlyBlocksWrites(t *testing.T) {
	app, registry := mongoConnectApp(t)

	resp, body := doJSON(t, app, http.MethodPost, "/api/connect", map[string]any{
		"url":      testMongoURI,
		"driver":   "mongodb",
		"readonly": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect: status=%d body=%v", resp.StatusCode, body)
	}
	sessionID, _ := body["sessionId"].(string)
	t.Cleanup(func() {
		if cl, err := registry.Get(sessionID); err == nil {
			_ = cl.Close(context.Background())
		}
	})

	connReq := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	connReq.Header.Set(sessionIDHeader, sessionID)
	connResp, err := app.Test(connReq)
	if err != nil {
		t.Fatalf("app.Test /api/connection: %v", err)
	}
	var info map[string]any
	if err := json.NewDecoder(connResp.Body).Decode(&info); err != nil {
		t.Fatalf("decode /api/connection: %v", err)
	}
	connResp.Body.Close()
	if ro, _ := info["readonly"].(bool); !ro {
		t.Errorf("expected session info readonly=true, got %v", info["readonly"])
	}

	// GET still works.
	listReq := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	listReq.Header.Set(sessionIDHeader, sessionID)
	listResp, err := app.Test(listReq)
	if err != nil {
		t.Fatalf("app.Test GET /api/databases: %v", err)
	}
	listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/databases: status = %d, want 200", listResp.StatusCode)
	}

	// A write on this session is rejected, even though the server itself
	// was not started with --readonly.
	createReq := httptest.NewRequest(http.MethodPost, "/api/databases", bytes.NewReader([]byte(`{"name":"should_not_be_created"}`)))
	createReq.Header.Set(sessionIDHeader, sessionID)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := app.Test(createReq)
	if err != nil {
		t.Fatalf("app.Test POST /api/databases: %v", err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /api/databases: status = %d, want 403", createResp.StatusCode)
	}
}

// TestAPIPerSessionReadonlyForcedWhenServerReadonly confirms a connect
// request can't downgrade a --readonly server's session to read-write by
// sending readonly:false — handleConnect ORs the request's value with the
// server-wide flag, so tampering with the connect form's HTML or calling
// the API directly can't produce a read-write session here.
func TestAPIPerSessionReadonlyForcedWhenServerReadonly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	registry := session.NewRegistry()
	app := New(Config{
		Registry: registry,
		Sessions: true,
		Readonly: true,
		Connect: func(ctx context.Context, driverName, uri string) (driver.Driver, error) {
			return client.Connect(ctx, uri)
		},
	})

	resp, body := doJSON(t, app, http.MethodPost, "/api/connect", map[string]any{
		"url":      testMongoURI,
		"driver":   "mongodb",
		"readonly": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect: status=%d body=%v", resp.StatusCode, body)
	}
	sessionID, _ := body["sessionId"].(string)
	t.Cleanup(func() {
		if cl, err := registry.Get(sessionID); err == nil {
			_ = cl.Close(context.Background())
		}
	})

	connReq := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	connReq.Header.Set(sessionIDHeader, sessionID)
	connResp, err := app.Test(connReq)
	if err != nil {
		t.Fatalf("app.Test /api/connection: %v", err)
	}
	var info map[string]any
	if err := json.NewDecoder(connResp.Body).Decode(&info); err != nil {
		t.Fatalf("decode /api/connection: %v", err)
	}
	connResp.Body.Close()
	if ro, _ := info["readonly"].(bool); !ro {
		t.Errorf("expected session info readonly=true (forced by server-wide flag), got %v", info["readonly"])
	}
}

func TestAPIExplainReturnsQueryPlan(t *testing.T) {
	app := newTestApp(t, false)

	if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/api_test_explain/collections", map[string]string{"name": "items"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	insertBody := []byte(`{"n":1}`)
	insertReq := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_explain/collections/items/documents", bytes.NewReader(insertBody))
	insertReq.Header.Set("Content-Type", "application/json")
	insertResp, err := app.Test(insertReq)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}
	insertResp.Body.Close()
	if insertResp.StatusCode != http.StatusCreated {
		t.Fatalf("insert document: status=%d", insertResp.StatusCode)
	}

	explainReq := httptest.NewRequest(http.MethodGet, `/api/databases/api_test_explain/collections/items/explain?filter=%7B%22n%22%3A1%7D`, nil)
	explainResp, err := app.Test(explainReq)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer explainResp.Body.Close()
	body, _ := io.ReadAll(explainResp.Body)
	if explainResp.StatusCode != http.StatusOK {
		t.Fatalf("explain: status=%d body=%s", explainResp.StatusCode, body)
	}
	if !bytes.Contains(body, []byte("queryPlanner")) {
		t.Errorf("expected queryPlanner in explain response, got: %s", body)
	}
}

func TestAPIAggregate(t *testing.T) {
	app := newTestApp(t, false)

	if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/api_test_agg/collections", map[string]string{"name": "items"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	for _, n := range []int{1, 2, 3} {
		insertBody, err := json.Marshal(map[string]int{"n": n})
		if err != nil {
			t.Fatalf("marshal insert body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_agg/collections/items/documents", bytes.NewReader(insertBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("insert document: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("insert document: status=%d", resp.StatusCode)
		}
	}

	pipeline := []byte(`[{"$group":{"_id":null,"total":{"$sum":"$n"}}}]`)
	aggReq := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_agg/collections/items/aggregate", bytes.NewReader(pipeline))
	aggReq.Header.Set("Content-Type", "application/json")
	aggResp, err := app.Test(aggReq)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	defer aggResp.Body.Close()
	body, _ := io.ReadAll(aggResp.Body)
	if aggResp.StatusCode != http.StatusOK {
		t.Fatalf("aggregate: status=%d body=%s", aggResp.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(`"total":6`)) {
		t.Errorf("expected total:6 in aggregate response, got: %s", body)
	}
}

func TestAPIAggregateRejectedInReadonlyMode(t *testing.T) {
	app := newTestApp(t, true)

	pipeline := []byte(`[{"$limit":1}]`)
	req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_agg_ro/collections/items/aggregate", bytes.NewReader(pipeline))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

// TestAPIUpdateMany mirrors the exact scenario asked for in practice:
// db.suaColecao.updateMany({ activity: null }, { $set: { activity: [] } }).
func TestAPIUpdateMany(t *testing.T) {
	app := newTestApp(t, false)

	if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/api_test_upd/collections", map[string]string{"name": "items"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	for _, doc := range []map[string]any{
		{"_id": "a", "activity": nil},
		{"_id": "b"},
		{"_id": "c", "activity": []string{"login"}},
	} {
		insertBody, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("marshal insert body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_upd/collections/items/documents", bytes.NewReader(insertBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("insert document: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("insert document: status=%d", resp.StatusCode)
		}
	}

	updateBody := []byte(`{"filter":{"activity":null},"update":{"$set":{"activity":[]}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_upd/collections/items/update-many", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update-many: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update-many: status=%d body=%s", resp.StatusCode, body)
	}
	if !bytes.Contains(body, []byte(`"matched":2`)) || !bytes.Contains(body, []byte(`"modified":2`)) {
		t.Errorf("expected matched:2 and modified:2 in response, got: %s", body)
	}

	getResp, getBody := doJSON(t, app, http.MethodGet, "/api/databases/api_test_upd/collections/items/documents?filter=%7B%22_id%22%3A%22c%22%7D", nil)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get c: status=%d body=%v", getResp.StatusCode, getBody)
	}
	docs, _ := getBody["documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected exactly 1 document for c, got %d: %v", len(docs), getBody)
	}
	c, _ := docs[0].(map[string]any)
	activity, _ := c["activity"].([]any)
	if len(activity) != 1 || activity[0] != "login" {
		t.Errorf("expected c.activity to remain [\"login\"] (not matched by the filter), got %v", c["activity"])
	}
}

func TestAPIUpdateManyRejectedInReadonlyMode(t *testing.T) {
	app := newTestApp(t, true)

	updateBody := []byte(`{"filter":{},"update":{"$set":{"x":1}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_upd_ro/collections/items/update-many", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestAPIToolsEndpoints(t *testing.T) {
	app := newTestApp(t, false)

	if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/api_test_tools/collections", map[string]string{"name": "items"}); resp.StatusCode != http.StatusCreated {
		t.Fatalf("create collection: status=%d body=%v", resp.StatusCode, body)
	}

	insertBody := []byte(`{"n":1}`)
	insertReq := httptest.NewRequest(http.MethodPost, "/api/databases/api_test_tools/collections/items/documents", bytes.NewReader(insertBody))
	insertReq.Header.Set("Content-Type", "application/json")
	insertResp, err := app.Test(insertReq)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}
	insertResp.Body.Close()
	if insertResp.StatusCode != http.StatusCreated {
		t.Fatalf("insert document: status=%d", insertResp.StatusCode)
	}

	overviewReq := httptest.NewRequest(http.MethodGet, "/api/databases/api_test_tools/tools/collections-overview", nil)
	overviewResp, err := app.Test(overviewReq)
	if err != nil {
		t.Fatalf("collections-overview: %v", err)
	}
	defer overviewResp.Body.Close()
	overviewBody, _ := io.ReadAll(overviewResp.Body)
	if overviewResp.StatusCode != http.StatusOK {
		t.Fatalf("collections-overview: status=%d body=%s", overviewResp.StatusCode, overviewBody)
	}
	if !bytes.Contains(overviewBody, []byte(`"name":"items"`)) {
		t.Errorf("expected items collection in overview, got: %s", overviewBody)
	}

	usageReq := httptest.NewRequest(http.MethodGet, "/api/databases/api_test_tools/tools/index-usage", nil)
	usageResp, err := app.Test(usageReq)
	if err != nil {
		t.Fatalf("index-usage: %v", err)
	}
	defer usageResp.Body.Close()
	usageBody, _ := io.ReadAll(usageResp.Body)
	if usageResp.StatusCode != http.StatusOK {
		t.Fatalf("index-usage: status=%d body=%s", usageResp.StatusCode, usageBody)
	}
	if !bytes.Contains(usageBody, []byte(`"_id_"`)) {
		t.Errorf("expected _id_ index in usage report, got: %s", usageBody)
	}

	opsReq := httptest.NewRequest(http.MethodGet, "/api/tools/current-ops?minSecs=3600", nil)
	opsResp, err := app.Test(opsReq)
	if err != nil {
		t.Fatalf("current-ops: %v", err)
	}
	defer opsResp.Body.Close()
	if opsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(opsResp.Body)
		t.Fatalf("current-ops: status=%d body=%s", opsResp.StatusCode, body)
	}
}

func TestAPIRunCommandMultiLineContinuesAfterError(t *testing.T) {
	app := newRedisTestApp(t, false)

	script := "SET foo bar\nGET foo\nINCR foo\nGET foo"
	body, err := json.Marshal(map[string]string{"script": script})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/databases/0/command", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, errBody)
	}

	var results []commandResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d: %+v", len(results), results)
	}
	if results[0].Result != "OK" {
		t.Errorf("SET: expected OK, got %+v", results[0])
	}
	if results[1].Result != "bar" {
		t.Errorf("GET: expected bar, got %+v", results[1])
	}
	if results[2].Error == "" {
		t.Errorf("INCR on a non-numeric string: expected an error, got %+v", results[2])
	}
	// The 4th line (after the failing 3rd) must still have run.
	if results[3].Result != "bar" {
		t.Errorf("final GET: expected the error on line 3 not to stop line 4, got %+v", results[3])
	}
}

func TestAPIRunCommandRejectedInReadonlyMode(t *testing.T) {
	app := newRedisTestApp(t, true)

	req := httptest.NewRequest(http.MethodPost, "/api/databases/0/command", bytes.NewReader([]byte(`{"script":"PING"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", resp.StatusCode)
	}
}
