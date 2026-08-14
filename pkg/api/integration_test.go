package api

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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

	"github.com/laubstein/mongo-express-go/pkg/client"
	"github.com/laubstein/mongo-express-go/pkg/session"
)

var testMongoURI string

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
	registry.Put(session.DefaultID, cl, session.Info{ID: session.DefaultID, Name: "test"})

	return New(Config{Registry: registry, Readonly: readonly})
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
