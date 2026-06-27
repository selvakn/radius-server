-- +goose Up
ALTER TABLE auth_attempts ADD COLUMN attempted_password TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; leave as-is on downgrade
