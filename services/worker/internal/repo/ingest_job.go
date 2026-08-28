package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IngestJobRepository interface {
	Create(ctx context.Context, job *model.IngestJob) (*model.IngestJob, error)
	Get(ctx context.Context, id uuid.UUID) (*model.IngestJob, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, docCount int, errMsg string) (*model.IngestJob, error)
}

type ingestJobRepository struct {
	pool *pgxpool.Pool
}

func NewIngestJobRepository(p *pgxpool.Pool) IngestJobRepository {
	return &ingestJobRepository{pool: p}
}

func (r *ingestJobRepository) Create(ctx context.Context, job *model.IngestJob) (*model.IngestJob, error) {
	query := `INSERT INTO ingest_jobs (session_id, user_id, status, source, source_type, doc_count, error)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, session_id, user_id, status, source, source_type, doc_count, error, created_at, updated_at`
	var j model.IngestJob
	if err := r.pool.QueryRow(ctx, query, job.SessionID, job.UserID, job.Status, job.Source, job.SourceType, job.DocCount, job.Error).Scan(
		&j.ID, &j.SessionID, &j.UserID, &j.Status, &j.Source, &j.SourceType, &j.DocCount, &j.Error, &j.CreatedAt, &j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *ingestJobRepository) Get(ctx context.Context, id uuid.UUID) (*model.IngestJob, error) {
	query := `SELECT id, session_id, user_id, status, source, source_type, doc_count, error, created_at, updated_at FROM ingest_jobs WHERE id = $1`
	var j model.IngestJob
	if err := r.pool.QueryRow(ctx, query, id).Scan(&j.ID, &j.SessionID, &j.UserID, &j.Status, &j.Source, &j.SourceType, &j.DocCount, &j.Error, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *ingestJobRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	query := `SELECT id, session_id, user_id, status, source, source_type, doc_count, error, created_at, updated_at FROM ingest_jobs WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []model.IngestJob
	for rows.Next() {
		var j model.IngestJob
		if err := rows.Scan(&j.ID, &j.SessionID, &j.UserID, &j.Status, &j.Source, &j.SourceType, &j.DocCount, &j.Error, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *ingestJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, docCount int, errMsg string) (*model.IngestJob, error) {
	query := `UPDATE ingest_jobs SET status = $2, doc_count = $3, error = $4 WHERE id = $1 RETURNING id, session_id, user_id, status, source, source_type, doc_count, error, created_at, updated_at`
	var j model.IngestJob
	if err := r.pool.QueryRow(ctx, query, id, status, docCount, errMsg).Scan(&j.ID, &j.SessionID, &j.UserID, &j.Status, &j.Source, &j.SourceType, &j.DocCount, &j.Error, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, err
	}
	return &j, nil
}
