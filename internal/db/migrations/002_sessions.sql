-- +goose Up
CREATE TABLE IF NOT EXISTS sessions (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT     UNIQUE NOT NULL,
    username        TEXT     NOT NULL,
    nas_ip          TEXT     NOT NULL DEFAULT '',
    started_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    stopped_at      DATETIME,
    bytes_in        INTEGER  NOT NULL DEFAULT 0,
    bytes_out       INTEGER  NOT NULL DEFAULT 0,
    session_time    INTEGER  NOT NULL DEFAULT 0,
    terminate_cause TEXT     NOT NULL DEFAULT '',
    status          TEXT     NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_sessions_username ON sessions(username);
CREATE INDEX IF NOT EXISTS idx_sessions_status   ON sessions(status);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_username;
DROP TABLE IF EXISTS sessions;
