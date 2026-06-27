-- +goose Up
ALTER TABLE sessions ADD COLUMN calling_station_id TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN in older versions; leave as-is on downgrade
