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

// Pins the fix for a real bug: embed.FS reports every file's ModTime as
// the zero time, identically across every build, so fasthttp's static
// handler would answer a conditional GET for "/" with a false 304 after
// a rebuild — pinning a browser tab to the previous build's index.html,
// which references JS/CSS filenames that no longer exist. "/" must always
// be served fresh (Cache-Control: no-store, and never a 304) regardless
// of what If-Modified-Since the client sends.
func TestAssetsIndexNeverServedFromCache(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>spa shell</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}

	app := New(Config{Registry: session.NewRegistry(), Assets: dist})

	for _, path := range []string{"/", "/index.html"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// The zero time, exactly as embed.FS would report it, and exactly
		// what a browser would echo back after caching a prior response.
		req.Header.Set("If-Modified-Since", "Mon, 01 Jan 0001 00:00:00 GMT")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s with stale If-Modified-Since: status = %d, want 200 (never 304)", path, resp.StatusCode)
		}
		if got := resp.Header.Get("Cache-Control"); got != "no-store" {
			t.Fatalf("GET %s: Cache-Control = %q, want %q", path, got, "no-store")
		}
	}
}
