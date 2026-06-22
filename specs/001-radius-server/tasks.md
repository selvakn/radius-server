# Tasks: RADIUS Server

**Input**: Design documents from `specs/001-radius-server/`  
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ contracts/ ✅

**Tests**: Included — constitution requires test-first TDD with ≥70% coverage. Write tests first, verify they fail, then implement.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story label — US1, US2, US3
- Tests are written BEFORE implementation (Red-Green-Refactor)

---

## Phase 1: Setup (Project Scaffold)

**Purpose**: Initialize Go module, Makefile, lint config, pre-commit hook.

- [ ] T001 Initialize Go module `github.com/selvakn/radius-server` and create directory structure per plan.md (`cmd/server/`, `internal/config/`, `internal/db/`, `internal/auth/`, `internal/web/templates/`)
- [ ] T002 Create `Makefile` with targets: `build` (single binary to `bin/radius-server`), `build-linux` (CGO_ENABLED=0 cross-compile), `test` (with -race -coverprofile), `lint` (golangci-lint), `run`, `clean`, `check` (lint then test — pre-commit gate)
- [ ] T003 [P] Create `.golangci.yml` enabling linters: `errcheck`, `staticcheck`, `govet`, `gofmt`, `revive`, `exhaustive` with zero-warnings policy
- [ ] T004 Add git pre-commit hook at `.git/hooks/pre-commit` that runs `make check` and exits non-zero on failure
- [ ] T005 [P] Create `config.yaml.example` with all fields documented, placeholder secrets, and two sample admin entries

**Commit after Phase 1**: `feat: initialize Go project scaffold with Makefile and lint config`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config loader and database layer — required by ALL user stories. No user story work can begin until this phase is complete.

**⚠️ CRITICAL**: Complete and commit this phase before starting Phase 3+.

### Config Loader

- [ ] T006 [P] Write failing tests for config loader in `internal/config/config_test.go`: load valid YAML, reject missing `radius.shared_secret`, reject missing `web.session_secret`, reject negative port, handle empty admins list gracefully
- [ ] T007 Implement `internal/config/config.go`: `Config` struct with `Radius`, `Database`, `Web`, `Admins` nested structs; `Load(path string) (*Config, error)` that reads YAML and validates required fields; fail-fast with descriptive error messages

### Database Layer

- [ ] T008 Create migration file `internal/db/migrations/001_initial.sql` with `users` table schema per data-model.md: id, username (UNIQUE), password_hash, enabled, download_rate (nullable), upload_rate (nullable), created_at, updated_at; add index on username
- [ ] T009 [P] Write failing tests for DB layer in `internal/db/users_test.go`: CreateUser success, duplicate username error, GetUserByUsername found/not-found, ListUsers, UpdateUser, SetEnabled toggle, DeleteUser, null rates handling
- [ ] T010 Implement `internal/db/db.go`: `Open(path string) (*DB, error)` that opens SQLite via `glebarez/sqlite`, runs embedded goose migration using `embed.FS`; add `go.mod` dependencies: `glebarez/sqlite`, `github.com/pressly/goose/v3`, `golang.org/x/crypto`, `layeh.com/radius`, `github.com/go-chi/chi/v5`, `gopkg.in/yaml.v3`
- [ ] T011 Implement `internal/db/users.go`: `User` struct; `CreateUser`, `GetUserByUsername`, `ListUsers`, `UpdateUser`, `SetEnabled`, `DeleteUser` — all using parameterized queries via `database/sql`

**Commit after Phase 2**: `feat: add config loader and SQLite database layer with goose migrations`

---

## Phase 3: User Story 1 — Authenticate PPPoE Dial-up Users (Priority: P1) 🎯 MVP

**Goal**: RADIUS server accepts Access-Request on UDP :1812 and responds with Access-Accept (+ MikroTik rate-limit VSA if configured) or Access-Reject based on user credentials and enabled status.

**Independent Test**: Start server with a test config, use `radtest` or a Go test client to send Access-Request with valid/invalid credentials, verify accept/reject responses and VSA attributes.

### Tests for US1 (Write FIRST — verify they FAIL before implementing)

- [ ] T012 [P] [US1] Write failing test `internal/auth/pap_test.go`: `TestVerifyPAP_ValidPassword` sends known User-Password attribute encrypted with test secret, verifies bcrypt match returns true; `TestVerifyPAP_WrongPassword` verifies false; `TestVerifyPAP_MalformedAttribute` verifies safe error handling
- [ ] T013 [P] [US1] Write failing test `internal/auth/handler_test.go`: `TestHandler_AcceptValidUser` creates test DB user, sends Access-Request via layeh/radius test client, asserts Access-Accept code; `TestHandler_RejectUnknownUser`; `TestHandler_RejectDisabledUser`; `TestHandler_RejectWrongPassword`; `TestHandler_IncludesRateLimitVSA` asserts Vendor 14988 Attr 8 present when user has rates set

### Implementation for US1

- [ ] T014 [US1] Implement `internal/auth/pap.go`: `VerifyPAP(req *radius.Request, secret, passwordHash string) bool` — decrypt User-Password per RFC 2865 §5.2 using MD5(secret+authenticator) XOR, then bcrypt compare
- [ ] T015 [US1] Implement `internal/auth/handler.go`: `Handler` struct with `DB *db.DB` and `Secret string`; `New(db, secret) *Handler`; `ServeRADIUS(w radius.ResponseWriter, r *radius.Request)` — lookup user, check enabled, call VerifyPAP, send Accept or Reject; `buildAccept` adds Framed-Protocol (PPP), optional MikroTik VSA; `mikrotikRateLimit(down, up int) radius.Attribute` encodes vendor 14988 attr 8 as `"<down>k/<up>k"` string
- [ ] T016 [US1] Add RADIUS server startup to `cmd/server/main.go`: parse `--config` flag, load config, open DB, create `auth.Handler`, start `radius.PacketServer` on UDP `:1812` with shared secret; add `log/slog` structured logging for accept/reject events

**Commit after Phase 3**: `feat: implement RADIUS authentication handler with PAP and MikroTik VSA`

**Checkpoint**: US1 fully functional — test with `radtest <username> <password> <server-ip> 1812 <secret>` or Go test.

---

## Phase 4: User Story 2 — Manage User Accounts via Admin UI (Priority: P1)

**Goal**: HTTP admin UI on :8080 allows authenticated admins to create, disable, enable, edit bandwidth limits, and permanently delete dial-up users. Changes take effect immediately on next RADIUS authentication.

**Independent Test**: Navigate to `http://localhost:8080`, log in, create a user, verify RADIUS accepts that user; disable the user, verify RADIUS rejects; set bandwidth limits, verify VSA in next Access-Accept.

### Tests for US2 (Write FIRST — verify they FAIL before implementing)

- [ ] T017 [P] [US2] Write failing test `internal/web/session_test.go`: `TestSessionCreate` verifies token is 32+ bytes hex, stored in map; `TestSessionGet` returns session for valid token; `TestSessionGet_Expired` returns not-found after TTL; `TestSessionDelete` removes entry
- [ ] T018 [P] [US2] Write failing test `internal/web/handlers_test.go`: `TestLogin_ValidCredentials` posts correct admin login, asserts 302 + Set-Cookie; `TestLogin_InvalidCredentials` asserts 401/re-render; `TestUsersIndex_Unauthenticated` asserts redirect to /login; `TestUsersIndex_Authenticated` asserts 200 with user table; `TestCreateUser_Success` posts new user form, asserts redirect and user exists in DB; `TestDisableUser` posts disable, asserts user.enabled=false; `TestDeleteUser` posts delete, asserts user removed; `TestCSRF_MissingToken` asserts 403

### Implementation for US2

- [ ] T019 [US2] Implement `internal/web/session.go`: `Store` struct with `sync.RWMutex` and `map[string]Session`; `Session{AdminUsername, ExpiresAt, CSRFToken}`; `Create(username string) (token string)`generates 32-byte crypto/rand hex token, stores with 8h expiry; `Get(token string) (*Session, bool)` checks expiry; `Delete(token string)`; background goroutine to prune expired sessions
- [ ] T020 [US2] Create HTML templates in `internal/web/templates/`: `layout.html` (base with flash message slot, logout link, nav); `login.html` (username+password form, hidden CSRF); `users.html` (dense table: username, status badge, rates, edit/disable-enable/delete buttons; new user form link at top)
- [ ] T021 [US2] Implement `internal/web/handlers.go`: `Handlers` struct with `*db.DB`, `*config.Config`, `*session.Store`; implement all route handlers per contracts/admin-web-ui.md — login/logout, list users, new user form, create user (bcrypt hash password), edit form, update user, set enabled, delete user; CSRF token generation and validation; flash message via signed cookie
- [ ] T022 [US2] Implement `internal/web/server.go`: `Server` struct; chi router setup; unauthenticated routes (`/login`); authenticated route group with session middleware and CSRF middleware; `//go:embed templates/*` for single-binary templates; `New(db, cfg, sessions) *Server`; `Start(port int) error`
- [ ] T023 [US2] Wire admin HTTP server into `cmd/server/main.go`: create session store, create web server, run RADIUS and HTTP servers concurrently via `errgroup` or goroutines; handle SIGINT/SIGTERM for graceful shutdown

**Commit after Phase 4**: `feat: implement admin web UI with chi router, session management, and user CRUD`

**Checkpoint**: US2 fully functional — log in at :8080, create/disable/delete users, verify RADIUS behavior changes immediately.

---

## Phase 5: User Story 3 — Configure Admin Access via File (Priority: P2)

**Goal**: Admin accounts defined in YAML config file control login access. Adding/removing an admin in the file and restarting the server grants or revokes UI access with no DB changes.

**Independent Test**: Add admin to config, restart, verify login works. Remove admin from config, restart, verify login is denied.

### Tests for US3 (Write FIRST — verify they FAIL before implementing)

- [ ] T024 [P] [US3] Write failing tests for admin auth in `internal/config/config_test.go`: `TestAdminUser_CheckPassword_Valid` verifies bcrypt compare returns true; `TestAdminUser_CheckPassword_Invalid` returns false; `TestConfig_FindAdmin_Found` and `TestConfig_FindAdmin_NotFound`; `TestConfig_Load_EmptyAdmins` succeeds with warning-level log
- [ ] T025 [P] [US3] Write failing test for hash-password subcommand in `cmd/server/main_test.go` or integration test: run binary with `hash-password` arg, pipe password to stdin, assert stdout is valid bcrypt hash

### Implementation for US3

- [ ] T026 [US3] Add to `internal/config/config.go`: `AdminUser` struct with `Username string` and `PasswordHash string` yaml tags; `(c *Config) FindAdmin(username string) (*AdminUser, bool)` linear scan of admins slice; `(a *AdminUser) CheckPassword(plain string) bool` calls `bcrypt.CompareHashAndPassword`
- [ ] T027 [US3] Add `hash-password` subcommand to `cmd/server/main.go`: if first arg is `hash-password`, read password from stdin (or prompt), print bcrypt hash (cost 12) to stdout, exit 0; document in `--help` output

**Commit after Phase 5**: `feat: implement config-file admin auth with hash-password subcommand`

**Checkpoint**: US3 fully functional — all three user stories independently verifiable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Integration verification, single-binary build, coverage check, final cleanup.

- [ ] T028 [P] Write integration test `internal/auth/integration_test.go` (build tag `integration`): starts full server with temp SQLite DB and test config, creates user via DB, sends actual UDP RADIUS Access-Request, asserts Access-Accept and rate-limit VSA
- [ ] T029 [P] Write integration test `internal/web/integration_test.go` (build tag `integration`): starts HTTP server, creates admin config, logs in via HTTP POST, asserts session cookie, creates user via form POST, asserts user in DB
- [ ] T030 Add graceful shutdown to `cmd/server/main.go`: `os/signal` + context cancellation; drain in-flight RADIUS requests; log "shutdown complete"
- [ ] T031 Verify single-binary build: run `CGO_ENABLED=0 make build-linux`, confirm binary runs on Linux without shared libs (`ldd bin/radius-server-linux-amd64` shows "not a dynamic executable")
- [ ] T032 Run `make check` (lint + test): fix any lint warnings (zero warnings policy per constitution); verify coverage ≥70% on changed lines
- [ ] T033 Update `quickstart.md` with any changes discovered during implementation (actual flags, real example commands)

**Final Commit**: `chore: add integration tests, verify single-binary build, lint and coverage passing`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 (needs `internal/db` and `internal/config`)
- **US2 (Phase 4)**: Depends on Phase 2; can start in parallel with US1 after Phase 2
- **US3 (Phase 5)**: Depends on Phase 2 (config) and Phase 4 (login handler must exist)
- **Polish (Phase 6)**: Depends on Phases 3, 4, 5 complete

### User Story Dependencies

- **US1 (P1)**: Independent after Phase 2 — RADIUS auth, no dependency on US2/US3
- **US2 (P1)**: Independent after Phase 2 — Admin UI, no dependency on US1/US3
- **US3 (P2)**: Depends on US2 (login handler) and Phase 2 (config loader)

### Within Each Phase

1. Write tests first (T###[a]) — verify they FAIL
2. Implement (T###[b]) — make tests pass
3. Run `make check` — lint + all tests green
4. Commit

---

## Parallel Opportunities

### Phase 2 Parallelizable

```bash
# These can run concurrently (different files):
Task T006: Write config tests (internal/config/config_test.go)
Task T009: Write DB tests (internal/db/users_test.go)
```

### Phase 3 Parallelizable

```bash
# Write these tests concurrently:
Task T012: Write PAP tests (internal/auth/pap_test.go)
Task T013: Write handler tests (internal/auth/handler_test.go)
```

### Phase 4 Parallelizable

```bash
# Write these tests concurrently:
Task T017: Write session tests (internal/web/session_test.go)
Task T018: Write handler tests (internal/web/handlers_test.go)

# Write templates concurrently with session store:
Task T019: Implement session store (internal/web/session.go)
Task T020: Create HTML templates (internal/web/templates/)
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (config + DB) — CRITICAL gate
3. Complete Phase 3: US1 (RADIUS authentication)
4. **STOP and VALIDATE**: `radtest admin pass 127.0.0.1 1812 secret` returns Access-Accept
5. Ship RADIUS-only binary; add admin UI later

### Incremental Delivery

1. Setup + Foundational → project builds, config loads, DB migrates
2. US1 → RADIUS authentication works with MikroTik ✅
3. US2 → Admin UI allows user management ✅
4. US3 → Config-file admin auth and hash-password helper ✅
5. Polish → integration tests, single-binary verified ✅

---

## Notes

- **Pre-commit gate**: `make check` must pass before EVERY commit (constitution Principle V)
- **Test-first**: Red-Green-Refactor — write test, verify FAIL, implement, verify PASS
- **Coverage**: ≥70% per constitution; `make test` generates `coverage.out` — check with `go tool cover -func=coverage.out`
- **File size limit**: No file exceeds 500 lines (constitution Principle III) — split if approaching limit
- **CHAP deferred**: PAP only in this implementation (CHAP requires plaintext password storage, conflicts with bcrypt)
- **Commits**: Frequent commits per user request; lint + tests must pass before each commit
