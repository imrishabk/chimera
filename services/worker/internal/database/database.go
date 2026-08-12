package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database interface {
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

func (db *database) CreateUser(ctx context.Context, username, email, passwordHash string) (*model.User, error) {
	query := `INSERT INTO users (username, email, password_hash)
	VALUES ($1, $2, $3)
	RETURNING id, username, email, password_hash, created_at, updated_at`

	var u model.User
	row := db.pool.QueryRow(ctx, query, username, email, passwordHash)
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *database) UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string) (*model.User, error) {
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

	var u model.User
	query := "UPDATE users SET "
	query += strings.Join(updates, ", ")
	query += fmt.Sprintf("WHERE id = $%d RETURNING id, username, email, password_hash, created_at, updated_at", counter)
	row := db.pool.QueryRow(ctx, query, queryArgs...)
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *database) FetchUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE id = $1`

	var u model.User
	row := db.pool.QueryRow(ctx, query, id)
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *database) FetchUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE username = $1`

	var u model.User
	row := db.pool.QueryRow(ctx, query, username)
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *database) FetchUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE email = $1`

	var u model.User
	row := db.pool.QueryRow(ctx, query, email)
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *database) CreateSession(ctx context.Context, userID uuid.UUID) (*model.Session, error) {
	query := `
	INSERT INTO sessions (user_id) VALUES ($1)
	RETURNING id, user_id, created_at
	`

	var s model.Session
	row := db.pool.QueryRow(ctx, query, userID)
	if err := row.Scan(&s.ID, &s.UserID, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *database) FetchSession(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	query := `
	SELECT id, user_id, created_at FROM sessions WHERE id = $1
	`

	var s model.Session
	row := db.pool.QueryRow(ctx, query, id)
	if err := row.Scan(&s.ID, &s.UserID, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *database) FetchSessionCount(ctx context.Context) (int, error) {
	query := `
	SELECT COUNT(*) FROM sessions
	`

	var count int
	row := db.pool.QueryRow(ctx, query)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (db *database) FetchSessionCountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
	SELECT count(*) FROM sessions WHERE user_id = $1
	`

	var count int
	row := db.pool.QueryRow(ctx, query, userID)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (db *database) ListSessionByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error) {
	query := `
	SELECT id, created_at FROM sessions WHERE user_id = $1
	LIMIT = $2 OFFSET = $3
	`

	rows, err := db.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var s []model.Session
	for rows.Next() {
		var r model.Session
		r.UserID = userID
		if err := rows.Scan(&r.ID, &r.CreatedAt); err != nil {
			return nil, err
		}
		s = append(s, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s, nil
}
