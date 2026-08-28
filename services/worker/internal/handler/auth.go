package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appErrors "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	"github.com/imrishabk/chimera/services/worker/internal/validator"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid request body"}`))
		return
	}

	if err := validator.Validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}

	u, err := h.svc.RegisterUser(r.Context(), &req)
	if err != nil {
		switch err.(type) {
		case appErrors.ValidationError:
			w.WriteHeader(http.StatusBadRequest)
		case appErrors.DatabaseError:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    u,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid request body"}`))
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	token, err := h.svc.LoginUser(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true,"data":{"token":"` + token + `"}}`))
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var req struct {
		Username string `json:"username" validate:"omitempty,username"`
		Email    string `json:"email" validate:"omitempty,email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid request body"}`))
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	uid, err := parseUUID(userID, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid userId"}`))
		return
	}
	u, err := h.svc.GetUser(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	// TODO: call UpdateUser service method when available; currently return fetched user
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": u})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-User-Token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}
	var req struct {
		ExpiresAt string `json:"expires_at"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	newToken, err := h.svc.RefreshUserSession(r.Context(), token, parseExpiry(req.ExpiresAt))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": map[string]string{"token": newToken}})
}

func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	uid, err := parseUUID(userID, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid userId"}`))
		return
	}
	if err := h.svc.LogoutFromAllDevice(r.Context(), uid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success":true,"data":"user sessions revoked"}`))
}

func parseUUID(s string, r *http.Request) (uuid.UUID, error) {
	if s == "" {
		s = chi.URLParam(r, "userId")
		if s == "" {
			s = chi.URLParam(r, "sessionId")
		}
	}
	return uuid.Parse(s)
}

func parseExpiry(s string) time.Time {
	if s == "" {
		return time.Now().Add(24 * time.Hour)
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now().Add(24 * time.Hour)
	}
	return t
}
