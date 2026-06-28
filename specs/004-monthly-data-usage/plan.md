# Implementation Plan: Monthly Data Usage

**Branch**: `004-monthly-data-usage` | **Date**: 2026-06-28 | **Spec**: [spec.md](spec.md)

## Summary

Aggregate bytes_in/bytes_out from the existing sessions table, grouped by calendar month in server local timezone. Show current-month totals on the users page (two new columns) and show a 24-month history table on the user edit page. Pure SQL aggregation — no new DB tables, no background jobs.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: existing stack only — no new libraries
**Storage**: SQLite — two new aggregate queries against `sessions` table using `strftime` + `datetime(..., 'localtime')`
**Testing**: `go test ./...`, TDD, ≥70% coverage
**Target Platform**: same binary
**Project Type**: additive feature — new DB functions + web handler updates + template changes
**Performance Goals**: <2s page load (SC-001, SC-002) at ≤100 users
**Constraints**: No new schema, no pre-computation, local timezone via SQLite `'localtime'` modifier

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Test-First ≥70% | ✅ | TDD for DB queries and handlers |
| II. Clean Code | ✅ | No unnecessary comments |
| III. ≤500 lines | ✅ | New files well under limit |
| IV. Minimal UI | ✅ | Two columns on users page, table below form |
| V. Pre-Commit Gates | ✅ | `make check` before commit |

**Gate result**: PASS.

## Project Structure

### Documentation

```text
specs/004-monthly-data-usage/
├── plan.md        ← this file
├── data-model.md
├── contracts/
│   └── usage-ui.md
└── tasks.md
```

### Source Changes

```text
internal/db/
└── usage.go           ← MonthlyUsage struct, GetCurrentMonthUsage, GetMonthlyUsageHistory
└── usage_test.go      ← tests

internal/web/
├── handlers.go        ← update handleGetUsers, handleGetEditUser
├── server.go          ← add MonthlyHistory and CurrentMonthUsage to pageData
└── templates/
    ├── users.html     ← add Upload / Download columns
    └── user_form.html ← add monthly history table
```

## SQL Design

### Current month usage for all users (users page)

```sql
SELECT username,
       SUM(bytes_in)  AS upload,
       SUM(bytes_out) AS download
FROM sessions
WHERE strftime('%Y-%m', datetime(
        CASE WHEN status = 'stopped' THEN stopped_at ELSE updated_at END,
        'localtime')) = strftime('%Y-%m', 'now', 'localtime')
GROUP BY username
```

Returns a row per user that has activity this month; absent users have zero.

### Monthly history for one user (user detail page)

```sql
SELECT strftime('%Y-%m', datetime(
         CASE WHEN status = 'stopped' THEN stopped_at ELSE updated_at END,
         'localtime')) AS month,
       SUM(bytes_in)  AS upload,
       SUM(bytes_out) AS download
FROM sessions
WHERE username = ?
GROUP BY month
ORDER BY month DESC
LIMIT 24
```

## Implementation Phases

### Phase 1: DB Layer

1. `internal/db/usage.go` — `MonthlyUsage` struct; `GetCurrentMonthUsage() (map[string]MonthlyUsage, error)`; `GetMonthlyUsageHistory(username string) ([]MonthlyUsage, error)`
2. Tests in `internal/db/usage_test.go`

**Commit**: `Add monthly usage DB queries`

### Phase 2: Web Layer

1. Add `CurrentMonthUsage map[string]MonthlyUsage` and `MonthlyHistory []MonthlyUsage` to `pageData` in `server.go`
2. Update `handleGetUsers` to fetch and pass current month usage
3. Update `handleGetEditUser` to fetch and pass monthly history
4. Add `fmtmonth` template func (converts "2026-06" → "Jun 2026")
5. Update `users.html` — add Upload / Download columns
6. Update `user_form.html` — add history table below the form

**Commit**: `Show monthly data usage in admin UI`

### Phase 3: Polish

1. `make check` — lint + tests ≥70%
2. `make build-linux`

**Commit**: `Lint clean for monthly usage`

## Key Dependencies (no new go.mod entries)

Uses existing: `database/sql`, `html/template`, `time`, `strconv`.
