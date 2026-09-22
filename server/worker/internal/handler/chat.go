package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	appValidator "github.com/imrishabk/chimera/services/worker/internal/validator"
)

type ChatHandler struct {
	svc service.ChatService
}

func NewChatHandler(svc service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) error {
	var req model.ChatRequest
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
	resp, err := h.svc.CreateChat(r.Context(), &req)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, resp)
}

func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) error {
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		sessionIDStr = chi.URLParam(r, "sessionId")
	}
	sid, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return appErrs.ErrInvalidSession
	}
	messages, err := h.svc.ListChats(r.Context(), sid)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, messages)
}

func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) error {
	var req model.ChatRequest
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

	stream, err := h.svc.ChatStream(r.Context(), &req)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		flusher = nil
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			data, _ := json.Marshal(map[string]interface{}{
				"done":       true,
				"session_id": req.SessionID.String(),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}
		if err != nil {
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}

		content := ""
		role := "assistant"
		if resp.GetMessage() != nil {
			content = resp.GetMessage().GetContent()
			if resp.GetMessage().GetRole() != "" {
				role = resp.GetMessage().GetRole()
			}
		}
		payload := map[string]interface{}{
			"token":      content,
			"content":    content,
			"role":       role,
			"done":       resp.GetDone(),
			"session_id": resp.GetSessionId(),
		}
		if payload["session_id"] == "" {
			payload["session_id"] = req.SessionID.String()
		}
		data, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
		if resp.GetDone() {
			return nil
		}
	}
}
