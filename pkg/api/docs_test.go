package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/laubstein/mongo-express-go/pkg/session"
)

func TestDocsServedUnderPrefixAndDoNotShadowAPI(t *testing.T) {
	docs := fstest.MapFS{
		"index.html":           {Data: []byte("<html>docs home</html>")},
		"getting-started.html": {Data: []byte("<html>getting started</html>")},
	}

	app := New(Config{Registry: session.NewRegistry(), Docs: docs})

	root := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	resp, err := app.Test(root)
	if err != nil {
		t.Fatalf("app.Test /doc/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /doc/: status = %d, want 200", resp.StatusCode)
	}

	page := httptest.NewRequest(http.MethodGet, "/doc/getting-started.html", nil)
	resp2, err := app.Test(page)
	if err != nil {
		t.Fatalf("app.Test /doc/getting-started.html: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /doc/getting-started.html: status = %d, want 200", resp2.StatusCode)
	}

	// /doc must not shadow /api: registerDocs is registered before the
	// SPA catch-all, but must still come after every /api route.
	infoReq := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	infoResp, err := app.Test(infoReq)
	if err != nil {
		t.Fatalf("app.Test /api/info: %v", err)
	}
	defer infoResp.Body.Close()
	if infoResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/info: status = %d, want 200", infoResp.StatusCode)
	}
}

func TestDocsNilIsNoop(t *testing.T) {
	app := New(Config{Registry: session.NewRegistry()})

	req := httptest.NewRequest(http.MethodGet, "/doc/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test /doc/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /doc/ with no Docs configured: status = %d, want 404", resp.StatusCode)
	}
}
