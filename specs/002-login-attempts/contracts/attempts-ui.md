# Contract: Login Attempts Admin UI

## Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/attempts` | session | Render attempt log page |
| GET | `/users/new?username=<name>` | session | New user form, username pre-filled |

## GET /attempts

Renders `attempts.html` within the layout. Queries `AttemptSummary` for up to 200 rows.

**Table columns**:
- Username
- 24h attempts (count, or `—` if zero)
- Last attempt (formatted timestamp)
- Last outcome (`accepted` / `rejected` badge)
- Status (`known` / `unknown` badge)
- Action (add button — shown only for unknown usernames)

**Empty state**: if no attempts exist, show "no authentication attempts recorded yet".

**Add button behaviour**: navigates to `GET /users/new?username=<encoded-username>`.

## GET /users/new?username=

Existing route extended with an optional `username` query parameter.

- If `?username=foo` is present, the username input field is pre-filled with `foo`
- If `foo` already exists in the database, the form submission redirects to `/users/<id>/edit` with a flash notice "User already exists"
- Behaviour without the query param is unchanged

## Nav bar

`attempts` link added between `users` and `sessions` in `layout.html`.
