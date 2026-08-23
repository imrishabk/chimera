-- +goose Up
CREATE TABLE user_sessions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  session TEXT UNIQUE NOT NULL,
  user_id UUID NOT NULL,
  CONSTRAINT fk_user_sessions_user_id
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_sessions_session ON user_sessions(session);

CREATE TRIGGER trg_user_sessions_updated_at
BEFORE UPDATE on user_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
SELECT 'down SQL query';
DROP TABLE user_sessions;
