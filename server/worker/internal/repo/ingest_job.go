package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type IngestJobRepository interface {
	Create(ctx context.Context, job *model.IngestJob) (*model.IngestJob, error)
	Get(ctx context.Context, id uuid.UUID) (*model.IngestJob, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, docCount int, errMsg string) (*model.IngestJob, error)
	// FindBySessionChecksum returns the latest non-failed job for identical
	// content, or ErrIngestionJobNotFound. Used for idempotent re-ingest.
	FindBySessionChecksum(ctx context.Context, sessionID uuid.UUID, checksum string) (*model.IngestJob, error)
}

type ingestJobRepository struct {
	pool *pgxpool.Pool
}

func NewIngestJobRepository(p *pgxpool.Pool) IngestJobRepository {
	return &ingestJobRepository{pool: p}
}

const ingestJobColumns = `id, session_id, user_id, status, source, source_type, doc_count, error, file_name, mime_type, file_size, checksum, created_at, updated_at`

func scanIngestJob(row pgx.Row) (*model.IngestJob, error) {
	var j model.IngestJob
	if err := row.Scan(
		&j.ID,
		&j.SessionID,
		&j.UserID,
		&j.Status,
		&j.Source,
		&j.SourceType,
		&j.DocCount,
		&j.Error,
		&j.FileName,
		&j.MimeType,
		&j.FileSize,
		&j.Checksum,
		&j.CreatedAt,
		&j.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *ingestJobRepository) Create(ctx context.Context, job *model.IngestJob) (*model.IngestJob, error) {
	query := `INSERT INTO ingest_jobs (session_id, user_id, status, source, source_type, doc_count, error, file_name, mime_type, file_size, checksum)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING ` + ingestJobColumns
	row := r.pool.QueryRow(ctx, query,
		job.SessionID,
		job.UserID,
		job.Status,
		job.Source,
		job.SourceType,
		job.DocCount,
		job.Error,
		job.FileName,
		job.MimeType,
		job.FileSize,
		job.Checksum)
	j, err := scanIngestJob(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, appErrs.ErrDuplicateIngestJob
		}
		return nil, &appErrs.DatabaseError{Operation: "IngestionJob: Create", Err: err}
	}
	return j, nil
}

func (r *ingestJobRepository) Get(ctx context.Context, id uuid.UUID) (*model.IngestJob, error) {
	query := `SELECT ` + ingestJobColumns + ` FROM ingest_jobs WHERE id = $1`
	j, err := scanIngestJob(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrs.ErrIngestionJobNotFound
		}
		return nil, &appErrs.DatabaseError{Operation: "IngestionJob: Get", Err: err}
	}
	return j, nil
}

func (r *ingestJobRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	query := `SELECT ` + ingestJobColumns + ` FROM ingest_jobs WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []model.IngestJob
	for rows.Next() {
		var j model.IngestJob
		if err := rows.Scan(&j.ID, &j.SessionID, &j.UserID, &j.Status, &j.Source, &j.SourceType, &j.DocCount, &j.Error, &j.FileName, &j.MimeType, &j.FileSize, &j.Checksum, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, &appErrs.DatabaseError{Operation: "IngestionJob: ListBySession", Err: err}
	}
	return jobs, nil
}

func (r *ingestJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, docCount int, errMsg string) (*model.IngestJob, error) {
	query := `UPDATE ingest_jobs SET status = $2, doc_count = $3, error = $4 WHERE id = $1 RETURNING ` + ingestJobColumns
	j, err := scanIngestJob(r.pool.QueryRow(ctx, query, id, status, docCount, errMsg))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrs.ErrIngestionJobNotFound
		}
		return nil, &appErrs.DatabaseError{Operation: "IngestionJob: UpdateStatus"}
	}
	return j, nil
}

func (r *ingestJobRepository) FindBySessionChecksum(ctx context.Context, sessionID uuid.UUID, checksum string) (*model.IngestJob, error) {
	if checksum == "" {
		return nil, appErrs.ErrIngestionJobNotFound
	}
	query := `SELECT ` + ingestJobColumns + ` FROM ingest_jobs
	WHERE session_id = $1 AND checksum = $2 AND status IN ('pending','processing','completed')
	ORDER BY created_at DESC LIMIT 1`
	j, err := scanIngestJob(r.pool.QueryRow(ctx, query, sessionID, checksum))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrs.ErrIngestionJobNotFound
		}
		return nil, &appErrs.DatabaseError{Operation: "IngestionJob: FindBySessionChecksum", Err: err}
	}
	return j, nil
}
