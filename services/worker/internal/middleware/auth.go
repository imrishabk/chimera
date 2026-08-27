package middleware

import (
	"net/http"
	"strings"

	"charm.land/log/v2"
)

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
