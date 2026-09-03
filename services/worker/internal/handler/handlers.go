package handler

import (
	"encoding/json"
	"net/http"

	"github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
)

// Handlers container holds interfaces for all route handlers.
// This allows main.go to initialize once and routes to receive the container.
type Handlers struct {
	Auth    *AuthHandler
	Session *SessionHandler
	Chat    *ChatHandler
	Ingest  *IngestHandler
	Query   *QueryHandler
}

func NewHandlers(svc *service.Services) *Handlers {
	h := &Handlers{
		Auth:    NewAuthHandler(svc.Auth),
		Session: NewSessionHandler(svc.Chat),
		Chat:    NewChatHandler(svc.Chat),
	}
	if svc.RAG != nil {
		if svc.IngestJob != nil {
			h.Ingest = NewIngestJobHandler(svc.IngestJob, svc.RAG)
		} else {
			h.Ingest = NewIngestHandler(svc.RAG)
		}
		h.Query = NewQueryHandler(svc.RAG)
	}
	return h
}

func writeJSONData[T any](w http.ResponseWriter, status int, data T) error {
	res := &model.APIResponse[T]{
		Success: true,
		Data:    data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(res)
	return nil
}

func writeJSONError(w http.ResponseWriter, err error) {
	status, errs := errors.StatusAndErrorMsg(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&model.APIErrorResponse{
		Success: false,
		Error:   errs,
	})
}

type AppHandler func(w http.ResponseWriter, r *http.Request) error

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		writeJSONError(w, err)
	}
}
