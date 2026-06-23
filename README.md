# radius-server

RADIUS authentication server for PPPoE dial-up users, with a minimal web-based admin interface. Built in Go, stores users in SQLite, ships as a single static binary.

## Features

- RADIUS Access-Request / Access-Accept / Access-Reject per RFC 2865, UDP port 1812
- PAP authentication with bcrypt-hashed passwords
- Per-user bandwidth limits sent as MikroTik Rate-Limit VSA and WISPr-Bandwidth-Max-Down/Up
- Message-Authenticator (RFC 3579) on every response
- Web admin UI: create, disable, enable, edit, delete users
- Admin accounts defined in YAML config — no database dependency
- Single statically-linked binary, no runtime dependencies

## Requirements

- Go 1.22+
- `golangci-lint` (for `make lint`)

## Configuration

Copy the example and edit:

```
cp config.yaml.example config.yaml
```

```yaml
radius:
  shared_secret: "your-secret"
  port: 1812

database:
  path: ./radius.db

web:
  port: 8080
  session_secret: "32-char-random-string"

admins:
  - username: admin
    password_hash: "$2a$12$..."
```

Generate a password hash:

```
./radius-server hash-password
```

## Build

```
make build           # bin/radius-server
make build-linux     # bin/radius-server-linux-amd64 (static, CGO_ENABLED=0)
```

## Run

```
make run             # go run with config.yaml
./radius-server --config config.yaml
```

## Docker

```
make docker-build
make docker-push
```

Or with compose:

```
docker compose up -d
```

The compose file expects `./config.yaml` to exist and mounts it read-only.

## Admin UI

Navigate to `http://localhost:8080` and log in with credentials from `config.yaml`.

Bandwidth rates are entered in Mbps (1–500). Values are stored internally as kbps and sent to the NAS in the RADIUS response.

## Testing authentication

```
radtest <username> <password> 127.0.0.1 0 <shared_secret>
```

Install `radtest` with:

```
sudo apt install freeradius-utils
```

## Development

```
make test    # run tests with race detector
make lint    # golangci-lint (errcheck, staticcheck, govet, revive, gosec)
make check   # lint + test (same gate as pre-commit hook)
```

Coverage is measured on each run; `coverage.out` is written to the project root.

## MikroTik setup

```
/radius add service=ppp address=<server-ip> secret=<shared_secret> port=1812
/ppp aaa set use-radius=yes
```
