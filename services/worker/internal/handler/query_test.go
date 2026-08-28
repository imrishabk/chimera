package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuerySubmitInvalidJSON(t *testing.T) {
	h := NewQueryHandler(&mockRAG{})
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}

func TestQuerySubmitValidationFail(t *testing.T) {
	h := NewQueryHandler(&mockRAG{})
	req := httptest.NewRequest(http.MethodPost, "/query", strings.NewReader(`{"query":""}`))
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}
