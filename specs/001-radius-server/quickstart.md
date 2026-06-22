# Quickstart: RADIUS Server

## Prerequisites

- Go 1.22+
- make
- golangci-lint (install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)

## Build

```bash
make build
# Output: ./bin/radius-server
```

Cross-compile for Linux:
```bash
make build-linux
# Output: ./bin/radius-server-linux-amd64
```

## Configure

```bash
cp config.yaml.example config.yaml
# Edit config.yaml:
# - Set radius.shared_secret
# - Set web.session_secret (32+ random chars)
# - Add admin users with bcrypt hashes
```

Generate admin password hash:
```bash
./bin/radius-server hash-password
```

## Run

```bash
make run
# or: ./bin/radius-server --config config.yaml
```

Server starts:
- RADIUS: UDP :1812
- Admin UI: HTTP :8080

## Development

```bash
make test    # run tests
make lint    # run golangci-lint
make check   # lint + test (pre-commit gate)
```

## Connect MikroTik

In MikroTik RouterOS:
```
/radius add service=ppp address=<server-ip> secret=<shared_secret> port=1812
/ppp aaa set use-radius=yes
```

## Admin UI

Navigate to `http://localhost:8080` — log in with credentials from config.yaml.
