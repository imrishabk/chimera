package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	appValidator "github.com/imrishabk/chimera/services/worker/internal/validator"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}
	if err := appValidator.Validate.Struct(req); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			return &appErrs.ValidationError{Fields: valErrs}
		}
		return err
	}
	u, err := h.svc.RegisterUser(r.Context(), &req)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, u)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrs.ErrInvalidBody
	}
	if err := appValidator.Validate.Struct(req); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			return &appErrs.ValidationError{Fields: valErrs}
		}
		return err
	}
	token, err := h.svc.LoginUser(r.Context(), &req)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	userID := chi.URLParam(r, "userId")
	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrs.ErrInvalidBody
	}
	if err := appValidator.Validate.Struct(req); err != nil {
		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			return &appErrs.ValidationError{Fields: valErrs}
		}
		return err
	}
	if req.OldPassword == req.NewPassword {
		return appErrs.ErrSamePassword
	}
	uid, err := parseUUID(userID, r)
	if err != nil {
		return appErrs.ErrInvalidUser
	}
	u, err := h.svc.UpdateUser(r.Context(), uid, &req)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, u)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	token := r.Header.Get("X-User-Token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}
	var req struct {
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrs.ErrInvalidBody
	}
	newToken, err := h.svc.RefreshUserSession(r.Context(), token, parseExpiry(req.ExpiresAt))
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, map[string]string{"token": newToken})
}

//	func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) error {
//		userID := chi.URLParam(r, "userId")
//		uid, err := parseUUID(userID, r)
//		if err != nil {
//			return &appErrs.HandlerError{Status: http.StatusBadRequest, Message: "invalid userId"}
//		}
//		// Ownership check: if token is present, ensure it belongs to target user
//		if token := extractToken(r); token != "" {
//			if ses, err := h.svc.ValidateToken(r.Context(), token); err == nil && ses.UserID != uid {
//				// w.Header().Set("Content-Type", "application/json")
//				// w.WriteHeader(http.StatusForbidden)
//				// w.Write([]byte(`{"success":false,"error":"cannot revoke sessions for another user"}`))
//				// return nil
//				//
//				return err
//			}
//		}
//		if err := h.svc.LogoutFromAllDevice(r.Context(), uid); err != nil {
//			return err
//		}
//		return writeJSONData(w, http.StatusOK, "deleted users")
//	}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	token := extractToken(r)
	if token == "" {
		return appErrs.ErrInvalidToken
	}
	if err := h.svc.Logout(r.Context(), token); err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, "logged out")
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) error {
	token := extractToken(r)
	if token == "" {
		return appErrs.ErrInvalidToken
	}
	if err := h.svc.LogoutFromAllDeviceByToken(r.Context(), token); err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, "all sessions removed")
}

func extractToken(r *http.Request) string {
	token := r.Header.Get("X-User-Token")
	if token == "" {
		token = r.Header.Get("Authorization")
	}
	if token == "" {
		token = r.Header.Get("X-Session-Token")
	}
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(token)
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
