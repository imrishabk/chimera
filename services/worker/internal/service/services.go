package service

import "github.com/imrishabk/chimera/services/worker/internal/repo"

type Services struct {
	Auth AuthService
	Chat ChatService
}

func NewServices(repos *repo.Repositories) *Services {
	return &Services{
		Auth: NewAuthService(repos.User, repos.UserSession),
		Chat: NewChatService(repos.User, repos.Session),
	}
}
