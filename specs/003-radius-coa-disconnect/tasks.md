# Tasks: RADIUS CoA — Admin Session Disconnect

**Input**: Design documents from `specs/003-radius-coa-disconnect/`
**Prerequisites**: plan.md ✅ spec.md ✅ data-model.md ✅ contracts/ ✅

**Tests**: Included — TDD, ≥70% coverage gate.

---

## Phase 1: CoA Disconnect Package

**Purpose**: Core RFC 5176 send logic. All other phases depend on this.

- [x] T001 [P] Write failing tests in `internal/coa/disconnect_test.go`: `TestSendDisconnect_ACK` starts a mock UDP server returning Disconnect-ACK, asserts nil error; `TestSendDisconnect_NAK` returns Disconnect-NAK, asserts error; `TestSendDisconnect_Timeout` uses cancelled context, asserts error
- [x] T002 Create `internal/coa/` package and implement `internal/coa/disconnect.go`: `SendDisconnect(ctx context.Context, nasIP, secret, sessionID, username string) error` — builds CodeDisconnectRequest packet with AcctSessionID + UserName, calls radius.Exchange to nasIP:3799, returns nil on CodeDisconnectACK, error on NAK or exchange error

**Commit after Phase 1**: `Add RFC 5176 CoA disconnect package`

---

## Phase 2: DB Queries (Priority: P1)

**Purpose**: Active session lookups needed by disconnect handlers.

- [x] T003 [P] Write failing tests in `internal/db/sessions_test.go`: `TestGetActiveSessionByID_Found`, `TestGetActiveSessionByID_NotFound`, `TestGetActiveSessionByID_InactiveReturnsNotFound`, `TestGetActiveSessionsByUser_ReturnsOnlyActive`
- [x] T004 Add to `internal/db/sessions.go`: `GetActiveSessionByID(id int64) (*Session, error)` (WHERE id = ? AND status = 'active'); `GetActiveSessionsByUser(username string) ([]Session, error)` (WHERE username = ? AND status = 'active')

**Commit after Phase 2**: `Add DB queries for active session lookup`

---

## Phase 3: Web Handlers + UI (Priority: P1)

**Purpose**: Admin-facing disconnect actions. Depends on Phases 1 and 2.

### Tests (write FIRST)

- [x] T005 [P] [US1] Write failing tests in `internal/web/handlers_test.go`: `TestDisconnectSession_NotFound` posts to unknown id, asserts flash error + redirect; `TestDisconnectSession_NoNasIP` creates session with no NAS IP, asserts flash error; `TestDisconnectSession_Success` mocks a successful disconnect (via injected CoA client), asserts session status updated and flash success
- [x] T006 [P] [US2] Write failing test: `TestDisconnectAll_NoActiveSessions` asserts flash "no active sessions"; `TestDisconnectAll_Success` with active sessions, asserts flash summary

### Implementation

- [x] T007 [US1] Implement `handlePostDisconnectSession` in `internal/web/handlers.go`: look up active session by id, validate NAS IP, call `s.coa.SendDisconnect` with 5s ctx, update session on ACK, set flash, redirect to `/sessions`
- [x] T008 [US2] Implement `handlePostDisconnectAllSessions` in `internal/web/handlers.go`: look up user, get active sessions, call SendDisconnect for each, flash summary, redirect to `/users/{id}/edit`
- [x] T009 Add `coa` field to `web.Server` struct and inject in `New()`; add routes `POST /sessions/{id}/disconnect` and `POST /users/{id}/disconnect-all` (both CSRF-protected) in `internal/web/server.go`
- [x] T010 [US1] Update `internal/web/templates/sessions.html`: add Disconnect column header; add disconnect form button per active row when NasIP is non-empty
- [x] T011 [US2] Update `internal/web/templates/user_form.html`: add "disconnect all sessions" form button below enable/disable controls

**Commit after Phase 3**: `Add session disconnect UI and handlers`

---

## Phase 4: Polish

- [x] T012 Run `make check`; fix any lint or coverage issues
- [x] T013 Run `CGO_ENABLED=0 make build-linux`; verify static binary

**Commit after Phase 4**: `Lint and coverage clean for CoA disconnect`

---

## Dependencies

- Phase 1: no dependencies — start immediately
- Phase 2: no dependencies — can run in parallel with Phase 1
- Phase 3: depends on Phases 1 and 2
- Phase 4: depends on Phase 3

## Notes

- NAS port 3799 is the RFC 5176 standard; hardcode in `SendDisconnect`
- No new go.mod entries needed
- CSRF on both POST routes (existing middleware)
- Disconnect button absent when session.NasIP == "" (template conditional)
