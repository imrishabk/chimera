package middleware

import (
	"net/http"
	"os"
	"strings"
)

// defaultDevOrigins covers the local frontend dev/preview servers.
var defaultDevOrigins = []string{
	"http://localhost:4321",
	"http://127.0.0.1:4321",
	"http://localhost:4333",
	"http://127.0.0.1:4333",
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return defaultDevOrigins
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(strings.TrimRight(p, "/")); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	// Dev convenience: any localhost port and private-LAN http origins.
	// Production should set CORS_ALLOWED_ORIGINS explicitly.
	rest, ok := strings.CutPrefix(origin, "http://")
	if !ok {
		return false
	}
	host := rest
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "localhost" || host == "127.0.0.1" {
		return true
	}
	if strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return true
	}
	for i := 16; i <= 31; i++ {
		if strings.HasPrefix(host, "172."+itoa(i)+".") {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [3]byte
	p := len(b)
	for n > 0 {
		p--
		b[p] = byte('0' + n%10)
		n /= 10
	}
	return string(b[p:])
}

// CORS is a dependency-free cross-origin middleware for local development.
// Apply it around the whole router (not via chi Use) so preflight OPTIONS
// is answered before chi's 405 handling:
//
//	handler := middleware.CORS(r)
//	http.ListenAndServe(":8000", handler)
//
// Origins come from CORS_ALLOWED_ORIGINS (comma-separated, "*" to allow all);
// defaults cover the Astro dev/preview ports. Requests without an Origin
// header (curl, same-origin) pass through untouched.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := allowedOrigins()
		if !originAllowed(origin, allowed) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Session-Token, X-User-Token, X-User-Id")
		w.Header().Set("Access-Control-Max-Age", "300")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
