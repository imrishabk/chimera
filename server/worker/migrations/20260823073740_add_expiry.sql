-- +goose Up
SELECT 'up SQL query';
ALTER TABLE user_sessions
ADD COLUMN expires_at TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '30 minutes');

ALTER TABLE user_sessions
ADD COLUMN expired BOOLEAN NOT NULL DEFAULT false;
-- +goose Down
SELECT 'down SQL query';
ALTER TABLE user_sessions
DROP COLUMN expires_at;
