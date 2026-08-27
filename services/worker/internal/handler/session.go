package handler

import (
	"net/http"

	"github.com/imrishabk/chimera/services/worker/internal/service"
)

type SessionHandler struct {
	svc service.ChatService
}

func NewSessionHandler(svc service.ChatService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true,"data":"Create session endpoint"}`))
}
