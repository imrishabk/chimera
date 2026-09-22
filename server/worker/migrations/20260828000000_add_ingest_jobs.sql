-- +goose Up
CREATE TABLE ingest_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','processing','completed','failed')) DEFAULT 'pending',
  source TEXT,
  source_type TEXT,
  doc_count INT NOT NULL DEFAULT 0,
  error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ingest_jobs_session_id ON ingest_jobs(session_id);
CREATE INDEX idx_ingest_jobs_user_id ON ingest_jobs(user_id);
CREATE TRIGGER trg_ingest_jobs_updated_at
BEFORE UPDATE ON ingest_jobs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_ingest_jobs_updated_at ON ingest_jobs;
DROP TABLE IF EXISTS ingest_jobs;
