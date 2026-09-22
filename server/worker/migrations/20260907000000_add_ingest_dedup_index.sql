-- +goose Up
-- Idempotent re-ingest guard: one completed job per (session, content hash).
-- Partial index so legacy rows with NULL/empty checksum are unaffected.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ingest_jobs_session_checksum
ON ingest_jobs (session_id, checksum)
WHERE checksum IS NOT NULL AND checksum <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_ingest_jobs_session_checksum;
