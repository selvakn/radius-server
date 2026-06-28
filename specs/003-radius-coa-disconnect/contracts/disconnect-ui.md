# Contract: Session Disconnect UI

## New Routes

| Method | Path | Auth | CSRF | Description |
|--------|------|------|------|-------------|
| POST | `/sessions/{id}/disconnect` | session | yes | Disconnect a single active session |
| POST | `/users/{id}/disconnect-all` | session | yes | Disconnect all active sessions for a user |

## POST /sessions/{id}/disconnect

**Input**: `{id}` = sessions.id (internal DB id)

**Flow**:
1. Look up session by id; if not found or not active → flash error, redirect to `/sessions`
2. If session has no NAS IP → flash error, redirect to `/sessions`
3. Call SendDisconnect(nasIP, secret, sessionID, username) with 5s timeout
4. On Disconnect-ACK → update session status to stopped, flash "Session disconnected", redirect to `/sessions`
5. On Disconnect-NAK → flash "NAS rejected disconnect", redirect to `/sessions`
6. On timeout/error → flash "Disconnect failed: <reason>", redirect to `/sessions`

## POST /users/{id}/disconnect-all

**Input**: `{id}` = users.id

**Flow**:
1. Look up user by id; if not found → 404
2. Fetch all active sessions for that user
3. If no active sessions → flash "No active sessions", redirect to `/users/{id}/edit`
4. For each session, call SendDisconnect; collect results
5. Flash summary: "Disconnected N of M sessions" (or "All N sessions disconnected")
6. Redirect to `/users/{id}/edit`

## UI Changes

### sessions.html

Add a disconnect column for active session rows:

```
| ... | Disconnect |
| ... | [disconnect] |  ← form POST /sessions/{id}/disconnect with CSRF
```

Button absent when `Status != "active"` or `NasIP == ""`.

### user_form.html

Add below the enable/disable buttons:

```
[disconnect all sessions]  ← form POST /users/{id}/disconnect-all with CSRF
```

Button present only when the user has active sessions (determined server-side; always rendered — server handles the "no active sessions" case gracefully).
