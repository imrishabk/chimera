package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*model.Session, error)
	FetchSession(ctx context.Context, id uuid.UUID) (*model.Session, error)
	FetchSessionCount(ctx context.Context) (int, error)
	FetchSessionCountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	ListSessionByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error)
}

type sessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(p *pgxpool.Pool) SessionRepository {
	return &sessionRepository{pool: p}
}

func (db *sessionRepository) CreateSession(ctx context.Context, userID uuid.UUID) (*model.Session, error) {
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

func (db *sessionRepository) FetchSession(ctx context.Context, id uuid.UUID) (*model.Session, error) {
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

func (db *sessionRepository) FetchSessionCount(ctx context.Context) (int, error) {
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

func (db *sessionRepository) FetchSessionCountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
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

func (db *sessionRepository) ListSessionByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error) {
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
