# Data Model: RADIUS CoA Session Disconnect

## No Schema Changes

No new database tables or columns are required. The existing `sessions` table already holds all data needed:

- `session_id` — sent as `Acct-Session-Id` in the Disconnect-Request
- `username` — sent as `User-Name` in the Disconnect-Request
- `nas_ip` — destination IP for the Disconnect-Request (port 3799)
- `status` — updated to `"stopped"` on Disconnect-ACK

## New DB Queries

**GetActiveSessionByID**: Fetch a single active session by internal DB id — used for per-session disconnect.

```sql
SELECT ... FROM sessions WHERE id = ? AND status = 'active'
```

**GetActiveSessionsByUser**: Fetch all active sessions for a username — used for bulk disconnect.

```sql
SELECT ... FROM sessions WHERE username = ? AND status = 'active'
```

## CoA Entities (in-memory only, not persisted)

**DisconnectRequest** (transient):
- `NasIP string` — from session.nas_ip
- `Secret string` — from server config (shared RADIUS secret)
- `SessionID string` — from session.session_id
- `Username string` — from session.username

**DisconnectResult** (transient):
- `OK bool` — true = Disconnect-ACK received
- `Err error` — network error, timeout, or Disconnect-NAK
