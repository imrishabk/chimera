-- +goose Up
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS file_name TEXT;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS mime_type TEXT;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS file_size BIGINT DEFAULT 0;
ALTER TABLE ingest_jobs ADD COLUMN IF NOT EXISTS checksum TEXT;

-- +goose Down
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS checksum;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS file_size;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS mime_type;
ALTER TABLE ingest_jobs DROP COLUMN IF EXISTS file_name;
