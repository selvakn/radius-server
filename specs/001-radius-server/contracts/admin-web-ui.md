# Contract: Admin Web UI (HTTP)

## Overview

Minimal server-side rendered HTML admin interface. All routes require authentication except `/login`. Default port: 8080 (configurable).

## Routes

### Authentication

| Method | Path | Description |
|--------|------|-------------|
| GET | `/login` | Render login form |
| POST | `/login` | Submit credentials, set session cookie, redirect to `/` |
| POST | `/logout` | Clear session cookie, redirect to `/login` |

### User Management (protected — require valid session)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | User list (table: username, status, bandwidth, actions) |
| GET | `/users/new` | New user form |
| POST | `/users` | Create user |
| GET | `/users/{id}/edit` | Edit user form (password optional, bandwidth limits) |
| POST | `/users/{id}` | Update user (method override: `_method=PUT` hidden field) |
| POST | `/users/{id}/disable` | Disable user |
| POST | `/users/{id}/enable` | Enable user |
| POST | `/users/{id}/delete` | Delete user (method override: `_method=DELETE`) |

## Form Fields

### Create/Edit User

| Field | Type | Validation | Notes |
|-------|------|------------|-------|
| `username` | text | Required, 1-64 chars | Read-only on edit |
| `password` | password | Required on create; optional on edit | Bcrypt-hashed server-side |
| `download_rate` | number | Optional, >0 | kbps; blank = no limit |
| `upload_rate` | number | Optional, >0 | kbps; blank = no limit |
| `_csrf` | hidden | Required | CSRF token |

## Session Cookie

| Property | Value |
|----------|-------|
| Name | `rsession` |
| HttpOnly | true |
| SameSite | Strict |
| Secure | false (operator adds TLS reverse proxy) |
| Expiry | 8 hours from last activity |

## UI Requirements (from Constitution IV — Minimal UI for Power Users)

- Single-page user list at `/` with inline status indicators
- No modals, no JavaScript frameworks — plain HTML forms
- Table-based layout: dense rows, minimal padding
- Action buttons per row: Edit, Disable/Enable, Delete
- Flash messages for success/error feedback (one-time, server-side)
- No confirmations dialogs — delete is immediate (by design for power users)

## Security

- CSRF token validated on all POST requests
- Session validated on all protected routes via middleware
- Admin credentials validated via bcrypt against config file
- No SQL injection: all queries use parameterized statements
