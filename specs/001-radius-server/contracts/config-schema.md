# Contract: Configuration File Schema (YAML)

## File Location

Default: `./config.yaml` (same directory as binary). Override via `--config` flag.

## Full Schema

```yaml
# RADIUS server configuration

radius:
  shared_secret: "change-me-in-production"  # Required. Shared secret for all NAS devices.
  port: 1812                                  # Optional. Default: 1812.

database:
  path: "./radius.db"                         # Optional. Default: ./radius.db

web:
  port: 8080                                  # Optional. Default: 8080.
  session_secret: "change-me-32-chars-min"   # Required. Random 32+ char string for session signing.

admins:
  - username: "admin"
    password_hash: "$2a$12$..."               # Required. bcrypt hash of admin password.
  - username: "operator"
    password_hash: "$2a$12$..."
```

## Field Reference

### `radius`
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `shared_secret` | string | — | **Required.** RADIUS shared secret for all connected NAS devices |
| `port` | integer | 1812 | UDP port for RADIUS authentication requests |

### `database`
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | `./radius.db` | Filesystem path to SQLite database file |

### `web`
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | integer | 8080 | TCP port for admin HTTP server |
| `session_secret` | string | — | **Required.** Secret key for signing session cookies (min 32 chars) |

### `admins` (list)
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `username` | string | — | **Required.** Admin login name |
| `password_hash` | string | — | **Required.** bcrypt hash of admin password |

## Generating Password Hashes

Use the binary's built-in helper:
```bash
./radius-server hash-password
# Enter password: ****
# $2a$12$...
```

Or via Apache htpasswd:
```bash
htpasswd -bnBC 12 "" yourpassword | tr -d ':\n'
```

## Error Handling

| Condition | Server Behavior |
|-----------|----------------|
| File not found | Fatal error: exits with message |
| Invalid YAML syntax | Fatal error: exits with parse error |
| Missing required field | Fatal error: exits with field name |
| Invalid port number | Fatal error: exits with range error |
| Empty admins list | Warning logged; admin UI inaccessible |
