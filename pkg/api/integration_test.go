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
	"net/url"
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

// TestAPIDatabasesListedAlphabetically covers handleListDatabases's sort:
// MongoDB's own listDatabases command returns server/catalog order, not
// alphabetical, so this only holds because the handler sorts explicitly.
func TestAPIDatabasesListedAlphabetically(t *testing.T) {
	app := newTestApp(t, false)

	names := []string{"zzz_test_order_db", "aaa_test_order_db", "mmm_test_order_db"}
	for _, name := range names {
		if resp, body := doJSON(t, app, http.MethodPost, "/api/databases", map[string]string{"name": name, "initialCollection": "seed"}); resp.StatusCode != http.StatusCreated {
			t.Fatalf("create database %q: status=%d body=%v", name, resp.StatusCode, body)
		}
		t.Cleanup(func() {
			doJSON(t, app, http.MethodDelete, "/api/databases/"+name, nil)
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/databases", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("list databases: %v", err)
	}
	defer resp.Body.Close()
	var all []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var seen []string
	for _, db := range all {
		for _, name := range names {
			if db.Name == name {
				seen = append(seen, db.Name)
			}
		}
	}
	if len(seen) != 3 {
		t.Fatalf("expected to find all 3 test databases in the response, found %v", seen)
	}
	if seen[0] != "aaa_test_order_db" || seen[1] != "mmm_test_order_db" || seen[2] != "zzz_test_order_db" {
		t.Errorf("databases not alphabetically ordered: %v", seen)
	}
}

// TestAPICollectionsListedAlphabetically covers handleListCollections's
// sort — this also fixes pkg/redis's collection list, which is otherwise
// built from a Go map and iterated in nondeterministic order.
func TestAPICollectionsListedAlphabetically(t *testing.T) {
	app := newTestApp(t, false)
	dbName := "api_test_coll_order"

	for _, name := range []string{"zzz", "aaa", "mmm"} {
		if resp, body := doJSON(t, app, http.MethodPost, "/api/databases/"+dbName+"/collections", map[string]string{"name": name}); resp.StatusCode != http.StatusCreated {
			t.Fatalf("create collection %q: status=%d body=%v", name, resp.StatusCode, body)
		}
	}
	t.Cleanup(func() {
		doJSON(t, app, http.MethodDelete, "/api/databases/"+dbName, nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/databases/"+dbName+"/collections", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("list collections: %v", err)
	}
	defer resp.Body.Close()
	var colls []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&colls); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(colls) != 3 {
		t.Fatalf("expected 3 collections, got %+v", colls)
	}
	if colls[0].Name != "aaa" || colls[1].Name != "mmm" || colls[2].Name != "zzz" {
		t.Errorf("collections not alphabetically ordered: %+v", colls)
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

// TestAPISessionSecretSignsAndValidatesSessionID covers --session-secret:
// with it set, /api/connect must return a signed token (not the raw
// registry key), requests carrying that token must work, and requests
// carrying the raw unsigned id (or a tampered token) must be rejected —
// see pkg/session.SignID/VerifyID and resolveSessionID.
func TestAPISessionSecretSignsAndValidatesSessionID(t *testing.T) {
	registry := session.NewRegistry()
	app := New(Config{
		Registry: registry,
		Sessions: true,
		Connect: func(ctx context.Context, driverName, uri string) (driver.Driver, error) {
			return client.Connect(ctx, uri)
		},
		SessionSecret: "test-secret",
	})

	resp, body := doJSON(t, app, http.MethodPost, "/api/connect", map[string]string{
		"url":    testMongoURI,
		"driver": "mongodb",
		"name":   "signed-session",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect: status=%d body=%v", resp.StatusCode, body)
	}
	token, _ := body["sessionId"].(string)

	rawID, ok := session.VerifyID("test-secret", token)
	if !ok {
		t.Fatalf("sessionId %q is not a validly signed token", token)
	}
	t.Cleanup(func() {
		if cl, err := registry.Get(rawID); err == nil {
			_ = cl.Close(context.Background())
		}
	})

	// The signed token works.
	req := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	req.Header.Set(sessionIDHeader, token)
	okResp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test /api/connection: %v", err)
	}
	defer okResp.Body.Close()
	if okResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/connection with signed token: status = %d, want 200", okResp.StatusCode)
	}

	// The raw, unsigned registry key must be rejected once a secret is set.
	rawReq := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	rawReq.Header.Set(sessionIDHeader, rawID)
	rawResp, err := app.Test(rawReq)
	if err != nil {
		t.Fatalf("app.Test /api/connection (raw id): %v", err)
	}
	defer rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /api/connection with raw unsigned id: status = %d, want 400", rawResp.StatusCode)
	}

	// A tampered token must be rejected too.
	tamperedReq := httptest.NewRequest(http.MethodGet, "/api/connection", nil)
	tamperedReq.Header.Set(sessionIDHeader, token+"tampered")
	tamperedResp, err := app.Test(tamperedReq)
	if err != nil {
		t.Fatalf("app.Test /api/connection (tampered): %v", err)
	}
	defer tamperedResp.Body.Close()
	if tamperedResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /api/connection with tampered token: status = %d, want 400", tamperedResp.StatusCode)
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

// exportTicketFor issues an export ticket through the normal scoped route
// and returns the token. sessionID may be empty for single-connection mode.
func exportTicketFor(t *testing.T, app *fiber.App, path, sessionID string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if sessionID != "" {
		req.Header.Set("X-Session-Id", sessionID)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("export ticket: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, string(raw)
	}
	var decoded struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode ticket response %q: %v", raw, err)
	}
	return resp.StatusCode, decoded.Ticket
}

// TestAPIExportHonorsSortAndProjection covers the export path end to end.
// The export used to receive only ?filter, so a projection that hid a
// field on screen still exported it — the file silently disagreed with
// what the user was looking at.
func TestAPIExportHonorsSortAndProjection(t *testing.T) {
	app := newTestApp(t, false)
	const db, coll = "api_test_export", "items"

	for _, doc := range []string{
		`{"name":"b","secret":"hidden","n":2}`,
		`{"name":"a","secret":"hidden","n":1}`,
	} {
		req := httptest.NewRequest(http.MethodPost,
			"/api/databases/"+db+"/collections/"+coll+"/documents", bytes.NewReader([]byte(doc)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("insert: status=%d", resp.StatusCode)
		}
	}
	t.Cleanup(func() { doJSON(t, app, http.MethodDelete, "/api/databases/"+db, nil) })

	base := "/api/databases/" + db + "/collections/" + coll + "/export"
	query := "?format=json&sort=" + url.QueryEscape(`{"n":1}`) + "&projection=" + url.QueryEscape(`{"secret":0}`)

	status, ticket := exportTicketFor(t, app, base+"/ticket"+query, "")
	if status != http.StatusOK {
		t.Fatalf("issue ticket: status=%d body=%s", status, ticket)
	}

	dlResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/export/"+ticket, nil))
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer dlResp.Body.Close()
	body, _ := io.ReadAll(dlResp.Body)
	if dlResp.StatusCode != http.StatusOK {
		t.Fatalf("download: status=%d body=%s", dlResp.StatusCode, body)
	}
	if bytes.Contains(body, []byte("secret")) {
		t.Errorf("projection was ignored, exported field is present: %s", body)
	}
	if a, b := bytes.Index(body, []byte(`"a"`)), bytes.Index(body, []byte(`"b"`)); a == -1 || b == -1 || a > b {
		t.Errorf("sort was ignored, expected a before b: %s", body)
	}

	// The ticket is single-use, so replaying a captured URL is inert.
	replay, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/export/"+ticket, nil))
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	defer replay.Body.Close()
	if replay.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a spent ticket to be rejected, got status=%d", replay.StatusCode)
	}
}

func TestAPIExportTicketRejectsBadInput(t *testing.T) {
	app := newTestApp(t, false)
	base := "/api/databases/api_test_export_bad/collections/items/export"

	// An invalid filter must fail at ticket time, so the UI can show it
	// inline instead of the browser navigating to a JSON error body.
	if status, body := exportTicketFor(t, app, base+"/ticket?format=json&filter=notjson", ""); status != http.StatusBadRequest {
		t.Errorf("expected 400 for an invalid filter, got status=%d body=%s", status, body)
	}
	if status, body := exportTicketFor(t, app, base+"/ticket?format=xml", ""); status != http.StatusBadRequest {
		t.Errorf("expected 400 for an unknown format, got status=%d body=%s", status, body)
	}

	unknown, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/export/does-not-exist", nil))
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer unknown.Body.Close()
	if unknown.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for an unknown ticket, got status=%d", unknown.StatusCode)
	}
}

// TestAPIExportWorksInSessionsMode is the regression test for the bug the
// ticket flow exists to fix: a browser download navigation cannot send
// X-Session-Id, and in --sessions mode there is no default session to fall
// back to, so the direct export route 400s.
func TestAPIExportWorksInSessionsMode(t *testing.T) {
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

	resp, body := doJSON(t, app, http.MethodPost, "/api/connect", map[string]string{
		"url": testMongoURI, "driver": "mongodb", "name": "export-session",
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

	base := "/api/databases/api_test_export_sessions/collections/items/export"

	// Without a header — exactly what a browser download does — the direct
	// route is unreachable in this mode.
	direct, err := app.Test(httptest.NewRequest(http.MethodGet, base+"?format=json", nil))
	if err != nil {
		t.Fatalf("direct export: %v", err)
	}
	defer direct.Body.Close()
	if direct.StatusCode != http.StatusBadRequest {
		t.Errorf("expected the headerless direct export to 400 in sessions mode, got %d", direct.StatusCode)
	}

	// The ticket is issued with the header and redeemed without one.
	status, ticket := exportTicketFor(t, app, base+"/ticket?format=json", sessionID)
	if status != http.StatusOK {
		t.Fatalf("issue ticket: status=%d body=%s", status, ticket)
	}
	dl, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/export/"+ticket, nil))
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer dl.Body.Close()
	if dl.StatusCode != http.StatusOK {
		out, _ := io.ReadAll(dl.Body)
		t.Errorf("ticketed export failed in sessions mode: status=%d body=%s", dl.StatusCode, out)
	}
}

// TestAPIExportWorksInReadonlyMode proves the ticket route has to be a GET:
// a POST would be rejected by rejectWrites, making export unusable in the
// mode where reading is the only thing you can do.
func TestAPIExportWorksInReadonlyMode(t *testing.T) {
	app := newTestApp(t, true)
	base := "/api/databases/api_test_export_ro/collections/items/export"

	status, ticket := exportTicketFor(t, app, base+"/ticket?format=csv", "")
	if status != http.StatusOK {
		t.Fatalf("issue ticket under --readonly: status=%d body=%s", status, ticket)
	}
	dl, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/export/"+ticket, nil))
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer dl.Body.Close()
	if dl.StatusCode != http.StatusOK {
		t.Errorf("expected export to work under --readonly, got status=%d", dl.StatusCode)
	}
}
