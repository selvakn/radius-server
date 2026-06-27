-- +goose Up
CREATE TABLE IF NOT EXISTS auth_attempts (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    username     TEXT     NOT NULL,
    attempted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    outcome      TEXT     NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_attempts_username     ON auth_attempts(username);
CREATE INDEX IF NOT EXISTS idx_auth_attempts_attempted_at ON auth_attempts(attempted_at);

-- +goose Down
DROP INDEX IF EXISTS idx_auth_attempts_attempted_at;
DROP INDEX IF EXISTS idx_auth_attempts_username;
DROP TABLE IF EXISTS auth_attempts;
