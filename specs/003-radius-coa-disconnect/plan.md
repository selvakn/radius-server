# Implementation Plan: RADIUS CoA — Admin Session Disconnect

**Branch**: `003-radius-coa-disconnect` | **Date**: 2026-06-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/003-radius-coa-disconnect/spec.md`

## Summary

Add admin-initiated session disconnect using RFC 5176 (Dynamic Authorization). A new `coa` package sends a Disconnect-Request UDP packet to the NAS on port 3799 using `radius.Exchange` with a 5-second context timeout. The sessions page gets a per-row disconnect button; the user edit page gets a "disconnect all" button. Results are delivered synchronously via flash messages.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: existing stack — `layeh.com/radius` already has `CodeDisconnectRequest/ACK/NAK` and `radius.Exchange`
**Storage**: SQLite — new DB queries only, no schema migration needed
**Testing**: `go test ./...` TDD; coverage gate ≥70%
**Target Platform**: same binary
**Project Type**: additive feature — new package + web handlers
**Performance Goals**: <5s synchronous response (SC-002)
**Constraints**: No new go.mod entries; NAS must be on UDP port 3799
**Scale/Scope**: low-frequency admin action, no concurrency concerns

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-First ≥70% | ✅ | TDD for coa package and handlers |
| II. Clean Code | ✅ | No unnecessary comments |
| III. ≤500 lines per file | ✅ | coa package is small |
| IV. Minimal UI | ✅ | One button per session row, one on edit page |
| V. Pre-Commit Gates | ✅ | `make check` before commit |

**Gate result**: PASS.

## Project Structure

### Documentation

```text
specs/003-radius-coa-disconnect/
├── plan.md         ← this file
├── data-model.md
├── contracts/
│   └── disconnect-ui.md
└── tasks.md
```

### Source Changes

```text
internal/coa/
├── disconnect.go        ← SendDisconnect(ctx, nasIP, secret, sessionID, username) error
└── disconnect_test.go   ← tests

internal/db/
└── sessions.go          ← GetActiveSessionByID, GetActiveSessionsByUser (new queries)

internal/web/
├── handlers.go          ← handlePostDisconnectSession, handlePostDisconnectAllSessions
├── server.go            ← new routes
└── templates/
    ├── sessions.html    ← disconnect button per active row
    └── user_form.html   ← disconnect all button
```

## Implementation Phases

### Phase 1: CoA Package

1. `internal/coa/disconnect.go` — `SendDisconnect(ctx, nasIP, secret, sessionID, username string) error`
   - Build `CodeDisconnectRequest` packet with `AcctSessionID` + `UserName`
   - `radius.Exchange(ctx, pkt, nasIP+":3799")`
   - Return nil on `CodeDisconnectACK`, error on NAK or timeout
2. Tests: ACK → nil, NAK → error, timeout → error

**Commit**: `Add RFC 5176 CoA disconnect package`

### Phase 2: DB Queries

1. `GetActiveSessionByID(id int64) (*Session, error)` — for single disconnect
2. `GetActiveSessionsByUser(username string) ([]Session, error)` — active only, for bulk disconnect

**Commit**: `Add DB queries for active session lookup by ID and user`

### Phase 3: Web Handlers + UI

1. `POST /sessions/{id}/disconnect` — look up session, call SendDisconnect, flash result, redirect
2. `POST /users/{id}/disconnect-all` — look up user's active sessions, call SendDisconnect for each, flash summary
3. CSRF protection on both routes
4. sessions.html: disconnect button on active rows with NAS IP
5. user_form.html: "disconnect all sessions" button

**Commit**: `Add session disconnect UI and handlers`

### Phase 4: Polish

1. `make check` lint + tests ≥70%
2. `make build-linux` static binary

**Commit**: `Lint and coverage clean for CoA disconnect`

## Key Dependencies (no new go.mod entries)

- `layeh.com/radius` — `radius.Exchange`, `radius.CodeDisconnectRequest/ACK/NAK`
- `layeh.com/radius/rfc2865` — `UserName_SetString`
- `layeh.com/radius/rfc2866` — `AcctSessionID_SetString`
