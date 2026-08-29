package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	"github.com/imrishabk/chimera/services/worker/internal/validator"
)

type ChatHandler struct {
	svc service.ChatService
}

func NewChatHandler(svc service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req model.ChatRequest
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
	resp, err := h.svc.CreateChat(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": resp})
}

func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		sessionIDStr = chi.URLParam(r, "sessionId")
	}
	sid, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid session_id"}`))
		return
	}
	messages, err := h.svc.ListChats(r.Context(), sid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": messages})
}

func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	var req model.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid request body"}`))
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}

	stream, err := h.svc.ChatStream(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
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
			return
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
			return
		}
		if err != nil {
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
			if flusher != nil {
				flusher.Flush()
			}
			return
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
			return
		}
	}
}
