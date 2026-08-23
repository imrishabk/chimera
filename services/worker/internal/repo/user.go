package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	CreateUser(ctx context.Context, username, email, passwordHash string) (*model.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string) (*model.User, error)
	FetchUser(ctx context.Context, id uuid.UUID) (*model.User, error)
	FetchUserByUsername(ctx context.Context, username string) (*model.User, error)
	FetchUserByEmail(ctx context.Context, email string) (*model.User, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(p *pgxpool.Pool) UserRepository {
	return &userRepository{pool: p}
}

func (db *userRepository) CreateUser(ctx context.Context, username, email, passwordHash string) (*model.User, error) {
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

func (db *userRepository) UpdateUser(ctx context.Context, id uuid.UUID, username, email, passwordHash string) (*model.User, error) {
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

func (db *userRepository) FetchUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
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

func (db *userRepository) FetchUserByUsername(ctx context.Context, username string) (*model.User, error) {
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

func (db *userRepository) FetchUserByEmail(ctx context.Context, email string) (*model.User, error) {
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
