package middleware

import (
	"encoding/json"
	"net/http"

	"charm.land/log/v2"

	appErrors "github.com/imrishabk/chimera/services/worker/internal/errors"
)

func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				w.Header().Set("Content-Type", "application/json")
				switch v := rec.(type) {
				case appErrors.ValidationError:
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   v.Error(),
					})
				case appErrors.DatabaseError:
					w.WriteHeader(http.StatusServiceUnavailable)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   v.Error(),
					})
				case appErrors.GRPCError:
					w.WriteHeader(http.StatusBadGateway)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   v.Error(),
					})
				case appErrors.HandlerError:
					w.WriteHeader(v.Status)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   v.Message,
					})
				default:
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   "internal server error",
					})
				}
				log.Error("Recovered from panic", "panic", rec)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
