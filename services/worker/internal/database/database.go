package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database interface {
	CreateUser(ctx context.Context, username, email, passwordHash string)
	UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string)
	GetUser(ctx context.Context, id uuid.UUID)
	GetUserFromUsername(ctx context.Context, username string)
	GetUserFromEmail(ctx context.Context, email string)

	CreateSession(ctx context.Context, userID uuid.UUID)
	GetSessionsFromUserID(ctx context.Context, userID uuid.UUID)
}

type database struct {
	pool *pgxpool.Pool
}

func NewDatabaseConnection(ctx context.Context, connString string) (Database, error) {
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
	return &database{pool}, nil
}

func (db *database) CreateUser(ctx context.Context, username, email, passwordHash string) {
	query := `INSERT INTO users (username, email, password_hash)
	VALUES ($1, $2, $3)
	RETURNING id, username, email, password_hash, created_at, updated_at`

	db.pool.QueryRow(ctx, query, username, email, passwordHash)
}

func (db *database) UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string) {
	var (
		updates   []string
		counter   = 1
		queryArgs []any
	)
	if username != "" {
		updates = append(updates, fmt.Sprintf("username = $%d", counter))
		counter += 1
		queryArgs = append(queryArgs, username)
	}
	if email != "" {
		updates = append(updates, fmt.Sprintf("email = $%d", counter))
		counter += 1
		queryArgs = append(queryArgs, email)
	}
	if passwordHash != "" {
		updates = append(updates, fmt.Sprintf("passwordHash = $%d", counter))
		counter += 1
		queryArgs = append(queryArgs, passwordHash)
	}
	query := "UPDATE users SET "
	query += strings.Join(updates, ", ")
	query += fmt.Sprintf("WHERE id = $%d", counter)
	db.pool.QueryRow(ctx, query, queryArgs...)
}

func (db *database) GetUser(ctx context.Context, id uuid.UUID) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE id = $1`
	db.pool.QueryRow(ctx, query, id)
}

func (db *database) GetUserFromUsername(ctx context.Context, username string) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE username = $1`
	db.pool.QueryRow(ctx, query, username)
}

func (db *database) GetUserFromEmail(ctx context.Context, email string) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE email = $1`
	db.pool.QueryRow(ctx, query, email)
}

func (db *database) CreateSession(ctx context.Context, userID uuid.UUID) {
	query := `
	INSERT INTO sessions (user_id) VALUES ($1)
	RETURNING id, user_id, created_at
	`
	db.pool.QueryRow(ctx, query, userID)
}

func (db *database) GetSession(ctx context.Context, id uuid.UUID) {
	query := `
	SELECT user_id, created_at FROM sessions WHERE id = $1
	`
	db.pool.QueryRow(ctx, query, id)
}

func (db *database) GetSessionsFromUserID(ctx context.Context, userID uuid.UUID) {
	query := `
	SELECT id, created_at FROM sessions WHERE user_id = $1
	`
	db.pool.Query(ctx, query, userID)
}
