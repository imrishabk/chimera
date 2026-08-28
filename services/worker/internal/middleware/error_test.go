package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	appErrors "github.com/imrishabk/chimera/services/worker/internal/errors"
)

func appErrorsValidation(field string) appErrors.ValidationError {
	return appErrors.ValidationError{Field: field, Message: "invalid"}
}
func appErrorsDatabase(op string) appErrors.DatabaseError {
	return appErrors.DatabaseError{Operation: op, Err: fmt.Errorf("db")}
}
func appErrorsGRPC(m string) appErrors.GRPCError {
	return appErrors.GRPCError{Method: m, Err: fmt.Errorf("grpc")}
}
func appErrorsHandler(status int) appErrors.HandlerError {
	return appErrors.HandlerError{Status: status, Message: "handler"}
}

func TestErrorHandlerRecoversPanic(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := ErrorHandler(panicHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	t.Log("Error middleware recovered panic and returned 500")
}

func TestErrorHandlerTypedPanics(t *testing.T) {
	tests := []struct {
		name       string
		panicVal   interface{}
		wantStatus int
	}{
		{"validation", appErrorsValidation("field"), http.StatusBadRequest},
		{"database", appErrorsDatabase("op"), http.StatusServiceUnavailable},
		{"grpc", appErrorsGRPC("method"), http.StatusBadGateway},
		{"handler", appErrorsHandler(418), 418},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(tc.panicVal) }))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			if w.Code != tc.wantStatus {
				t.Errorf("%s: want %d got %d", tc.name, tc.wantStatus, w.Code)
			}
		})
	}
}
