-- +goose Up
ALTER TABLE users ADD COLUMN nt_hash TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; leave as-is on downgrade
