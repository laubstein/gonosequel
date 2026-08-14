package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/laubstein/gonosequel/pkg/session"
)

func TestAssetsServesIndexAndSPAFallback(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>spa shell</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}

	app := New(Config{Registry: session.NewRegistry(), Assets: dist})

	root := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(root)
	if err != nil {
		t.Fatalf("app.Test /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /: status = %d, want 200", resp.StatusCode)
	}

	asset := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	resp2, err := app.Test(asset)
	if err != nil {
		t.Fatalf("app.Test /assets/app.js: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/app.js: status = %d, want 200", resp2.StatusCode)
	}

	deep := httptest.NewRequest(http.MethodGet, "/some/client/side/route", nil)
	resp3, err := app.Test(deep)
	if err != nil {
		t.Fatalf("app.Test deep route: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("GET /some/client/side/route: status = %d, want 200 (SPA fallback)", resp3.StatusCode)
	}
}
