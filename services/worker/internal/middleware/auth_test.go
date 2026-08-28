package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddlewareMissingToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := AuthMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddlewareBearerToken(t *testing.T) {
	var capturedToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedToken = r.Header.Get("X-User-Token")
		w.WriteHeader(http.StatusOK)
	})
	h := AuthMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer mytoken123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedToken != "mytoken123" {
		t.Errorf("expected token mytoken123, got %s", capturedToken)
	}
}

func TestAuthMiddlewareXSessionToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := AuthMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Session-Token", "sess-abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for X-Session-Token, got %d", w.Code)
	}
}
