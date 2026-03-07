package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"

	_ "github.com/scc749/nimbus-blog-api/docs"
)

func TestSwaggerRoutes(t *testing.T) {
	app := fiber.New()
	app.Get("/swagger/*", swaggo.HandlerDefault)

	paths := []string{
		"/swagger/index.html",
		"/swagger/doc.json",
	}

	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %s failed: %v", p, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			t.Fatalf("unexpected status for %s: %d", p, resp.StatusCode)
		}
	}
}
