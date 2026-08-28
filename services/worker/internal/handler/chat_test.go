package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatSendInvalidJSON(t *testing.T) {
	h := NewChatHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	h.Send(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}

func TestChatSendValidationFail(t *testing.T) {
	h := NewChatHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"input":""}`))
	w := httptest.NewRecorder()
	h.Send(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing session_id, got %d", w.Code)
	}
}

func TestChatListInvalidSession(t *testing.T) {
	h := NewChatHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodGet, "/chat?session_id=bad", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}
