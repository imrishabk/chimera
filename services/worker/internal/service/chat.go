package service

import (
	"github.com/imrishabk/chimera/services/worker/internal/repo"
)

type ChatService interface{}

type chatService struct {
	user repo.UserRepository
}
