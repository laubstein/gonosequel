package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/gonosequel/pkg/session"
)

func TestInfoEndpoint(t *testing.T) {
	app := New(Config{Registry: session.NewRegistry()})

	req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

// TestBasicAuthDoesNotPanicOnSetup is a regression test for a startup
// panic ("decode SHA256 password: illegal base64 data at input byte 1"):
// fiber v3's basicauth.Config.Users holds a *hash* of the password, not
// the plaintext — passing AuthPass straight through panicked in New()
// (via basicauth.New -> configDefault -> buildVerifiers) as soon as
// --auth-user/--auth-pass were set, since it tried to decode the
// plaintext as a hex/base64-encoded SHA-256 digest.
func TestBasicAuthDoesNotPanicOnSetup(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New() panicked with AuthUser/AuthPass set: %v", r)
		}
	}()
	New(Config{Registry: session.NewRegistry(), AuthUser: "admin", AuthPass: "s3cr3t"})
}

func TestBasicAuthRejectsMissingOrWrongCredentials(t *testing.T) {
	app := New(Config{Registry: session.NewRegistry(), AuthUser: "admin", AuthPass: "s3cr3t"})

	req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no credentials: status = %d, want 401", resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.SetBasicAuth("admin", "wrong-password")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: status = %d, want 401", resp.StatusCode)
	}
}

func TestBasicAuthAcceptsCorrectCredentials(t *testing.T) {
	app := New(Config{Registry: session.NewRegistry(), AuthUser: "admin", AuthPass: "s3cr3t"})

	req := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	req.SetBasicAuth("admin", "s3cr3t")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("correct credentials: status = %d, want 200", resp.StatusCode)
	}
}

func TestReadonlyRejectsWrites(t *testing.T) {
	app := New(Config{Readonly: true, Registry: session.NewRegistry()})
	app.Post("/api/_test-write", func(c fiber.Ctx) error { return nil })

	req := httptest.NewRequest(http.MethodPost, "/api/_test-write", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}
