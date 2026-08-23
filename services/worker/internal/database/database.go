package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateUser(ctx context.Context, username, email, passwordHash string) (*model.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string) (*model.User, error)
	FetchUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	FetchUserByUsername(ctx context.Context, username string) (*model.User, error)
	FetchUserByEmail(ctx context.Context, email string) (*model.User, error)

	CreateSession(ctx context.Context, userID uuid.UUID) (*model.Session, error)
	FetchSession(ctx context.Context, id uuid.UUID) (*model.Session, error)
	FetchSessionCount(ctx context.Context) (int, error)
	FetchSessionCountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	ListSessionByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error)
}

func NewDatabaseConnection(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 5 * time.Minute
	config.MaxConnLifetimeJitter = 3 * time.Minute
	config.MaxConnLifetime = 10 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
