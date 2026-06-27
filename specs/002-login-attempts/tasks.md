# Tasks: Login Attempt Log

**Input**: Design documents from `specs/002-login-attempts/`
**Prerequisites**: plan.md ✅ spec.md ✅ data-model.md ✅ contracts/ ✅

**Tests**: Included — constitution requires test-first TDD with ≥70% coverage.

**Organization**: Tasks grouped by phase for sequential delivery.

---

## Phase 1: Setup (Migration + DB Layer)

**Purpose**: New `auth_attempts` table and all query functions. Required before any other phase.

- [x] T001 Create migration `internal/db/migrations/003_auth_attempts.sql` with `auth_attempts` table (username, attempted_at, outcome) and two indexes per data-model.md
- [x] T002 [P] Write failing tests in `internal/db/attempts_test.go`: `TestRecordAttempt`, `TestListAttemptSummaries_Count24h`, `TestListAttemptSummaries_LastOutcome`, `TestListAttemptSummaries_IsKnown`, `TestListAttemptSummaries_Cap200`, `TestPurgeOldAttempts`
- [x] T003 Implement `internal/db/attempts.go`: `AttemptSummary` struct; `RecordAttempt(username, outcome string) error`; `ListAttemptSummaries() ([]AttemptSummary, error)` (aggregate query: 24h count, last time, last outcome, is_known join, LIMIT 200, ORDER BY last_attempted_at DESC); `PurgeOldAttempts() error` (DELETE WHERE attempted_at < 7 days ago)

**Commit after Phase 1**: `Add auth_attempts table and DB layer`

---

## Phase 2: User Story 1 — Record Attempts in Auth Handler (Priority: P1)

**Goal**: Every RADIUS authentication attempt (accept or reject) is persisted to `auth_attempts`.

**Independent Test**: Send RADIUS Access-Requests with known and unknown usernames, query `auth_attempts` table directly, verify rows exist with correct outcome and timestamp.

### Tests for Phase 2 (write FIRST)

- [x] T004 [P] [US1] Write failing test in `internal/auth/handler_test.go`: `TestHandler_RecordsAcceptAttempt` — after accepting a valid user, assert a row exists in `auth_attempts` with outcome `"accepted"`; `TestHandler_RecordsRejectAttempt` — after rejecting unknown user, assert row with outcome `"rejected"`

### Implementation for Phase 2

- [x] T005 [US1] Update `internal/auth/handler.go`: add `*db.DB` field to `Handler`; update `New(database, secret)` signature; call `db.RecordAttempt(username, "accepted"/"rejected")` after each auth decision (log error but never fail the RADIUS response)
- [x] T006 [US1] Update `cmd/server/main.go`: pass `database` to `auth.New`; add `startPurgeLoop(database)` goroutine that calls `db.PurgeOldAttempts()` once at startup and then every hour

**Commit after Phase 2**: `Record auth attempts in RADIUS handler`

**Checkpoint**: `radtest` with known and unknown usernames; verify rows appear in the DB.

---

## Phase 3: User Story 2 — Attempts Page + One-Click Provision (Priority: P1)

**Goal**: Admin sees `/attempts` page with all attempt summaries. Unknown usernames have an "add" button that pre-fills the new user form. Duplicate username on submit redirects to edit page with notice.

**Independent Test**: Navigate to `/attempts`, verify table columns (username, 24h count, last time, last outcome, status, action). Click "add" on an unknown username, verify form is pre-filled. Submit; verify user created and RADIUS accepts.

### Tests for Phase 3 (write FIRST)

- [x] T007 [P] [US2] Write failing test in `internal/web/handlers_test.go`: `TestGetAttempts_Authenticated` asserts 200; `TestGetAttempts_Unauthenticated` asserts redirect to /login; `TestGetAttempts_EmptyState` asserts empty state message when no attempts; `TestGetAttempts_AddButtonOnlyForUnknown` seeds one known and one unknown attempt, asserts add button HTML present only for unknown row
- [x] T008 [P] [US2] Write failing test: `TestGetNewUser_UsernameQueryParam` asserts that `GET /users/new?username=bob` renders with `bob` pre-filled in the username input; `TestCreateUser_DuplicateRedirectsToEdit` asserts that posting a duplicate username redirects to `/users/<id>/edit` with a flash notice

### Implementation for Phase 3

- [x] T009 [US2] Create `internal/web/templates/attempts.html` with `{{define "content"}}` block: table with columns username, 24h count, last attempt time, last outcome badge, status badge (known/unknown), add button link for unknown rows; empty state row when no data
- [x] T010 [US2] Add `handleGetAttempts` handler in `internal/web/handlers.go`: query `db.ListAttemptSummaries()`, render `attempts.html`; add route `r.Get("/attempts", s.handleGetAttempts)` in `server.go`
- [x] T011 [US2] Update `handleGetNewUser` in `internal/web/handlers.go` to read `?username=` query param and pass it into the form data so the template pre-fills the username field
- [x] T012 [US2] Update `handlePostCreateUser` in `internal/web/handlers.go`: on duplicate username error, look up the existing user by username, redirect to `/users/<id>/edit` with flash notice "User already exists"
- [x] T013 [US2] Add `attempts` link to nav bar in `internal/web/templates/layout.html` (between users and sessions)
- [x] T014 [US2] Update `internal/web/templates/user_form.html` so the username input renders with a pre-filled value when provided (currently always empty for new users)

**Commit after Phase 3**: `Add /attempts admin page with one-click user provisioning`

**Checkpoint**: Full end-to-end — send attempts via `radtest`, view `/attempts`, click add, provision user, verify RADIUS accepts.

---

## Phase 4: Polish

**Purpose**: Lint clean, coverage gate, binary build verified.

- [x] T015 Run `make check` (lint + tests); fix any issues until zero warnings and coverage ≥70%
- [x] T016 Run `CGO_ENABLED=0 make build-linux`; verify static binary builds cleanly

**Commit after Phase 4**: `Lint and coverage clean for login attempt log`

---

## Dependencies & Execution Order

- **Phase 1**: No dependencies — start immediately
- **Phase 2**: Depends on Phase 1 (needs `attempts.go` functions)
- **Phase 3**: Depends on Phase 1 (needs `ListAttemptSummaries`); can start in parallel with Phase 2 for the UI template work (T009)
- **Phase 4**: Depends on Phases 2 and 3 complete

## Notes

- `RecordAttempt` must never block or fail a RADIUS response — log errors, don't propagate
- Pre-commit hook runs `make check` automatically; don't commit with failing tests or lint
- `AttemptSummary.IsKnown` is determined by a LEFT JOIN or EXISTS subquery against `users` at query time — not stored
