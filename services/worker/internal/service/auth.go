package service

import (
	"github.com/imrishabk/chimera/services/worker/internal/repo"
)

type AuthService interface{}

type authService struct {
	user repo.UserRepository
}
