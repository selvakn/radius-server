# Data Model: Login Attempt Log

## New Entity: AuthAttempt

Stores one row per individual RADIUS authentication attempt.

**SQLite table: `auth_attempts`**

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal identifier |
| `username` | TEXT | NOT NULL | Username from the Access-Request |
| `attempted_at` | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP | When the attempt occurred |
| `outcome` | TEXT | NOT NULL | `"accepted"` or `"rejected"` |

**Indexes**:
- `idx_auth_attempts_username` on `username` — supports aggregate queries
- `idx_auth_attempts_attempted_at` on `attempted_at` — supports purge and 24h window queries

**Retention**: rows with `attempted_at` older than 7 days are purged automatically.

---

## Derived View: AttemptSummary

Not a stored table — computed by SQL aggregate query at page load.

**Query output per unique username**:

| Field | Source |
|-------|--------|
| `username` | `GROUP BY username` |
| `count_24h` | `COUNT(*) WHERE attempted_at >= NOW - 24h` |
| `last_attempted_at` | `MAX(attempted_at)` |
| `last_outcome` | outcome of the row with `MAX(attempted_at)` |
| `is_known` | `EXISTS (SELECT 1 FROM users WHERE username = ...)` |

**Ordering**: `last_attempted_at DESC LIMIT 200`

---

## Migration

File: `internal/db/migrations/003_auth_attempts.sql`

```sql
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
```
