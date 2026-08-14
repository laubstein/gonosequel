package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/laubstein/mongo-express-go/pkg/session"
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
