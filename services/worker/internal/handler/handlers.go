package handler

import (
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
