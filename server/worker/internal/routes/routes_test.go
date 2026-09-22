package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imrishabk/chimera/services/worker/internal/handler"
	"github.com/imrishabk/chimera/services/worker/internal/service"
)

func setupRouter() http.Handler {
	svc := &service.Services{}
	h := handler.NewHandlers(svc)
	return Configure(svc, h)
}

func TestHealthRoute(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	t.Log("Health route responds with 200")
}

func TestAuthRegisterRouteExists(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Errorf("auth register route should exist, got 404")
	}
	t.Logf("Auth register responds with status %d", w.Code)
}
