package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
	"github.com/imrishabk/chimera/services/worker/internal/util"
)

type AuthService interface {
	LoginUser(c context.Context, r *model.LoginRequest) (string, error)
	RegisterUser(c context.Context, r *model.RegisterRequest) (*model.User, error)
	GetUser(c context.Context, userID uuid.UUID) (*model.User, error)
	RefreshUserSession(c context.Context, token string, expiresAt time.Time) (string, error)
	Logout(c context.Context, token string) error
	LogoutFromAllDevice(c context.Context, userID uuid.UUID) error
}

type authService struct {
	user        repo.UserRepository
	userSession repo.UserSessionRepository
}

func NewAuthService(userRepo repo.UserRepository, sessionRepo repo.UserSessionRepository) AuthService {
	return &authService{user: userRepo, userSession: sessionRepo}
}

func (s *authService) LoginUser(c context.Context, r *model.LoginRequest) (string, error) {
	u, err := s.user.FetchUserByUsername(c, r.Username)
	if err != nil {
		return "", fmt.Errorf("invalid Username or Password")
	}
	if !util.VerifyPassword(r.Password, u.PasswordHash) {
		return "", fmt.Errorf("invalid Username or Password")
	}
	token, err := util.CreateUserSession()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if _, err := s.userSession.RegisterToken(c, token, u.ID, expiresAt, false); err != nil {
		return "", err
	}
	return token, nil
}

func (s *authService) RegisterUser(c context.Context, r *model.RegisterRequest) (*model.User, error) {
	if _, err := s.user.FetchUserByUsername(c, r.Username); err == nil {
		return nil, fmt.Errorf("username already exists")
	}
	if _, err := s.user.FetchUserByEmail(c, r.Email); err == nil {
		return nil, fmt.Errorf("email already exists")
	}
	passwordHash, err := util.HashPassword(r.Password)
	if err != nil {
		return nil, err
	}
	u, err := s.user.CreateUser(c, r.Username, r.Email, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (s *authService) GetUser(c context.Context, userID uuid.UUID) (*model.User, error) {
	u, err := s.user.FetchUser(c, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return u, nil
}

func (s *authService) RefreshUserSession(c context.Context, token string, expiresAt time.Time) (string, error) {
	ses, err := s.userSession.FetchToken(c, token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	if ses.Expired || time.Now().After(ses.ExpiresAt) {
		return "", fmt.Errorf("Token already expired")
	}
	if _, err := s.userSession.SetSessionExpiredByToken(c, token); err != nil {
		return "", err
	}
	if expiresAt.IsZero() || expiresAt.Before(time.Now()) {
		expiresAt = time.Now().Add(24 * time.Hour)
	}
	t, err := util.CreateUserSession()
	if err != nil {
		return "", err
	}
	if _, err := s.userSession.RegisterToken(c, t, ses.UserID, expiresAt, false); err != nil {
		return "", err
	}
	return t, nil
}

func (s *authService) Logout(c context.Context, token string) error {
	_, err := s.userSession.SetSessionExpiredByToken(c, token)
	return err
}

func (s *authService) LogoutFromAllDevice(c context.Context, userID uuid.UUID) error {
	_, err := s.userSession.SetSessionExpiredByUserID(c, userID)
	return err
}
