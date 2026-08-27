package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
