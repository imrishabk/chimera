package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSessionRepository interface {
	FetchToken(ctx context.Context, token string) (*model.UserSession, error)
	RegisterToken(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, expired bool) (*model.UserSession, error)
	SetSessionExpiredByToken(ctx context.Context, token string) (*model.UserSession, error)
	SetSessionExpiredByUserID(ctx context.Context, userID uuid.UUID) (*model.UserSession, error)
}

type userSessionRepository struct {
	pool *pgxpool.Pool
}

func (db *userSessionRepository) FetchToken(ctx context.Context, token string) (*model.UserSession, error) {
	query := `
	SELECT id, token, user_id, created_at, updated_at, expires_at, expired
	FROM user_sessions WHERE token = $1
	`
	var ses model.UserSession
	if err := db.pool.QueryRow(ctx, query, token).Scan(
		&ses.ID,
		&ses.Token,
		&ses.UserID,
		&ses.CreatedAt,
		&ses.UpdatedAt,
		&ses.ExpiresAt,
		&ses.Expired,
	); err != nil {
		return nil, err
	} else {
		return &ses, err
	}
}

func (db *userSessionRepository) RegisterToken(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, expired bool) (*model.UserSession, error) {
	query := `
	INSERT INTO user_sessions (token, user_id, expires_at, expired)
	VALUES ($1, $2, $3, $4)
	RETURNING id, token, user_id, created_at, updated_at, expires_at, expired;
	`
	var ses model.UserSession
	if err := db.pool.QueryRow(ctx, query, token, userID, expiresAt, expiresAt).Scan(
		&ses.ID,
		&ses.Token,
		&ses.UserID,
		&ses.CreatedAt,
		&ses.ExpiresAt,
		&ses.Expired,
	); err != nil {
		return nil, err
	} else {
		return &ses, nil
	}
}

func (db *userSessionRepository) SetSessionExpiredByToken(ctx context.Context, token string) (*model.UserSession, error) {
	query := `
	UPDATE user_sessions SET expired = true, expired_at = now()
	WHERE token = $1
	RETURNING id, token, user_id, created_at, updated_at, expires_at, expired
	`
	var ses model.UserSession
	if err := db.pool.QueryRow(ctx, query, token).Scan(
		&ses.ID,
		&ses.Token,
		&ses.UserID,
		&ses.CreatedAt,
		&ses.ExpiresAt,
		&ses.Expired,
	); err != nil {
		return nil, err
	} else {
		return &ses, nil
	}
}

func (db *userSessionRepository) SetSessionExpiredByUserID(ctx context.Context, userID uuid.UUID) (*model.UserSession, error) {
	query := `
	UPDATE user_sessions SET expired = true, expired_at = now()
	WHERE user_id = $1
	RETURNING id, token, user_id, created_at, updated_at, expires_at, expired
	`
	var ses model.UserSession
	if err := db.pool.QueryRow(ctx, query, userID).Scan(
		&ses.ID,
		&ses.Token,
		&ses.UserID,
		&ses.CreatedAt,
		&ses.ExpiresAt,
		&ses.Expired,
	); err != nil {
		return nil, err
	} else {
		return &ses, nil
	}
}
