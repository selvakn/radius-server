# Data Model: RADIUS Server

## Entities

### User

Represents a dial-up (PPPoE) account that can authenticate via RADIUS.

**SQLite table: `users`**

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal identifier |
| `username` | TEXT | UNIQUE NOT NULL | PPPoE login name |
| `password_hash` | TEXT | NOT NULL | bcrypt hash of password |
| `enabled` | INTEGER | NOT NULL DEFAULT 1 | 1 = active, 0 = disabled |
| `download_rate` | INTEGER | NULL | Max download in kbps; NULL = no limit |
| `upload_rate` | INTEGER | NULL | Max upload in kbps; NULL = no limit |
| `created_at` | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| `updated_at` | DATETIME | NOT NULL DEFAULT CURRENT_TIMESTAMP | Last modification timestamp |

**Validation rules**:
- `username`: 1–64 characters, alphanumeric + hyphen + underscore + dot
- `password_hash`: bcrypt, cost factor ≥ 10
- `download_rate`, `upload_rate`: if set, must be > 0 (kbps)

**State transitions**:
```
enabled=1 (active) ──[disable]──► enabled=0 (disabled)
enabled=0 (disabled) ──[enable]──► enabled=1 (active)
enabled=* ──[delete]──► (row removed)
```

**Authentication logic**:
1. Lookup by `username` — if not found → Access-Reject
2. Check `enabled = 1` — if 0 → Access-Reject
3. Verify password (PAP: bcrypt compare; CHAP: MD5 challenge/response) — if fail → Access-Reject
4. If pass → Access-Accept + optional rate-limit attributes

---

### Admin (Config-Defined)

Administrator accounts are **not stored in the database**. They are defined in the YAML configuration file and loaded at startup.

**Go struct** (from config):
```go
type AdminUser struct {
    Username     string `yaml:"username"`
    PasswordHash string `yaml:"password_hash"` // bcrypt
}
```

**Session** (in-memory only):
```go
type Session struct {
    AdminUsername string
    ExpiresAt     time.Time
    CSRFToken     string
}
```
Sessions map: `map[sessionToken]Session` in server memory. Sessions expire after 8 hours of inactivity or on server restart.

---

### Bandwidth Profile (embedded in User)

Not a separate entity — stored as `download_rate` + `upload_rate` columns in the `users` table.

**MikroTik RADIUS encoding**:
- When both rates are non-NULL, include VSA in Access-Accept response:
  - Vendor ID: `14988` (MikroTik)
  - Attribute: `8` (Mikrotik-Rate-Limit)
  - Value: `"<down>k/<up>k"` — e.g., `"2048k/1024k"` for 2Mbps down / 1Mbps up

---

## Migration

Single migration file embedded in the binary:

```sql
-- migrations/001_initial.sql
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT    UNIQUE NOT NULL,
    password_hash   TEXT    NOT NULL,
    enabled         INTEGER NOT NULL DEFAULT 1,
    download_rate   INTEGER,
    upload_rate     INTEGER,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
```

Migration is applied on startup using goose or a simple `PRAGMA user_version` check.
