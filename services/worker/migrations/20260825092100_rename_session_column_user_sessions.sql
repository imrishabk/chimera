-- +goose Up
SELECT 'up SQL query';
ALTER TABLE user_sessions
RENAME COLUMN session TO token;

-- +goose Down
SELECT 'down SQL query';
