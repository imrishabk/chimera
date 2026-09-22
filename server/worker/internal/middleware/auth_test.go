package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"charm.land/log/v2"
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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			authHeader = r.Header.Get("X-Session-Token")
		}

		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success":false,"error":"Authorization token required"}`))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		var token string
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = parts[1]
		} else {
			token = authHeader
		}

		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"success":false,"error":"Invalid authorization format"}`))
			return
		}

		r.Header.Set("X-User-Token", token)
		log.Info("Auth middleware passed", "token_prefix", token[:min(8, len(token))])
		next.ServeHTTP(w, r)
	})
}
