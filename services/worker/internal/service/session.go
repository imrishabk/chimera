package service

import "github.com/imrishabk/chimera/services/worker/internal/repo"

type SessionService interface{}

type sessionService struct {
	session repo.SessionRepository
}
