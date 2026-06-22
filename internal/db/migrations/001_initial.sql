-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    username        TEXT     UNIQUE NOT NULL,
    password_hash   TEXT     NOT NULL,
    enabled         INTEGER  NOT NULL DEFAULT 1,
    download_rate   INTEGER,
    upload_rate     INTEGER,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- +goose Down
DROP INDEX IF EXISTS idx_users_username;
DROP TABLE IF EXISTS users;
