# Tasks: Monthly Data Usage

**Input**: Design documents from `specs/004-monthly-data-usage/`
**Prerequisites**: plan.md ✅ spec.md ✅ data-model.md ✅ contracts/ ✅

**Tests**: TDD — write tests first, ≥70% coverage.

---

## Phase 1: DB Layer

- [ ] T001 [P] Write failing tests in `internal/db/usage_test.go`: `TestGetCurrentMonthUsage_WithData`, `TestGetCurrentMonthUsage_NoData`, `TestGetMonthlyUsageHistory_MultipleMonths`, `TestGetMonthlyUsageHistory_Cap24`
- [ ] T002 Implement `internal/db/usage.go`: `MonthlyUsage` struct; `GetCurrentMonthUsage() (map[string]MonthlyUsage, error)`; `GetMonthlyUsageHistory(username string) ([]MonthlyUsage, error)`

**Commit**: `Add monthly usage DB queries`

---

## Phase 2: Web Layer

- [ ] T003 [P] Write failing tests: `TestUsersPage_ShowsCurrentMonthUsage`, `TestEditUser_ShowsMonthlyHistory`
- [ ] T004 Add `CurrentMonthUsage map[string]db.MonthlyUsage` and `MonthlyHistory []db.MonthlyUsage` to `pageData` in `server.go`; add `fmtmonth` template func
- [ ] T005 Update `handleGetUsers` in `handlers.go` to query `GetCurrentMonthUsage` and pass to template
- [ ] T006 Update `handleGetEditUser` in `handlers.go` to query `GetMonthlyUsageHistory` and pass to template
- [ ] T007 Update `users.html`: add Upload and Download columns showing current month usage
- [ ] T008 Update `user_form.html`: add monthly history table below form

**Commit**: `Show monthly data usage in admin UI`

---

## Phase 3: Polish

- [ ] T009 `make check` (lint + tests ≥70%)
- [ ] T010 `make build-linux` (static binary)

**Commit**: `Lint clean for monthly usage`
