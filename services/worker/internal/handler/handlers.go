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
}

func NewHandlers(svc *service.Services) *Handlers {
	return &Handlers{
		Auth:    NewAuthHandler(svc.Auth),
		Session: NewSessionHandler(svc.Chat),
		Chat:    NewChatHandler(svc.Chat),
	}
}
