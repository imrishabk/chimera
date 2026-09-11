package middleware

import (
	"net/http"
	"strings"
)

// JSONContentType returns a middleware that sets the Content-Type header
// to "application/json; charset=utf-8" for all requests, except those
// whose path contains one of the given excludePaths
func JSONContentType(excludePaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range excludePaths {
				if strings.Contains(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			next.ServeHTTP(w, r)
		})
	}
}
