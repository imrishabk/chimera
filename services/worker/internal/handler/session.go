package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	appValidator "github.com/imrishabk/chimera/services/worker/internal/validator"
)

type SessionHandler struct {
	svc service.ChatService
}

func NewSessionHandler(svc service.ChatService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		UserID uuid.UUID `json:"user_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrs.ErrInvalidBody
	}
	if req.UserID == uuid.Nil {
		if uidStr := chi.URLParam(r, "userId"); uidStr != "" {
			if uid, err := uuid.Parse(uidStr); err == nil {
				req.UserID = uid
			}
		}
	}
	if err := appValidator.Validate.Struct(req); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			return &appErrs.ValidationError{Fields: valErrs}
		}
		return err
	}
	ses, err := h.svc.CreateSession(r.Context(), req.UserID)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, ses)
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) error {
	userIDStr := chi.URLParam(r, "userId")
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return appErrs.ErrInvalidUser
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	sessions, err := h.svc.ListSessions(r.Context(), uid, limit, offset)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, sessions)
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) error {
	sessionIDStr := chi.URLParam(r, "sessionId")
	sid, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return appErrs.ErrInvalidSession
	}
	ses, err := h.svc.GetSession(r.Context(), sid)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, ses)
}
