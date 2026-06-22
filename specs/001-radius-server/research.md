# Research: RADIUS Server (Go)

## Decision 1: RADIUS Library

**Decision**: Use `layeh.com/radius` (github.com/layeh/radius)

**Rationale**: Pure Go implementation, no CGO, compatible with single-binary distribution. Provides RFC 2865 packet parsing, attribute encoding/decoding, and vendor-specific attribute (VSA) support. Only maintained Go RADIUS library with production usage.

**Alternatives considered**:
- `blind-oracle/go-radius`: A fork with more features but less maintained.
- Raw UDP + manual RFC 2865 parsing: Viable but unnecessary effort given layeh/radius coverage.

**CHAP implementation**: `layeh/radius` provides attribute helpers (CHAP-Password attr 3, CHAP-Challenge attr 60) but NOT server-side verification logic. Must implement manually: `MD5(CHAP_ID || plaintext_password || challenge)` and compare against the client's CHAP-Password value. This is ~10 lines of code.

**MikroTik vendor-specific attributes**:
- Vendor ID: `14988`
- `Mikrotik-Rate-Limit` (Attribute 8): String format, e.g. `"1024k/512k"` (down/up). This is the primary attribute MikroTik reads for queue rate limiting.
- Encoding: Standard VSA encoding per RFC 2865 §5.26 via `radius.AddVendorSpecific`.

---

## Decision 2: SQLite Driver

**Decision**: Use `modernc.org/sqlite` via the `glebarez/sqlite` database/sql wrapper

**Rationale**: Pure Go, zero CGO dependency. True single-binary cross-compilation with `GOOS=linux GOARCH=amd64 go build`. Performance overhead (~5-15% vs mattn) is negligible for this workload (max 100 users, low concurrent auth load).

**Alternatives considered**:
- `mattn/go-sqlite3`: Battle-tested and fast, but requires CGO + gcc at build time. Breaks cross-compilation for Linux targets from macOS/Windows. **Rejected** — violates single-binary requirement.
- `zombiezen.com/go/sqlite`: Another pure-Go option but less ecosystem adoption.

**Migrations**: Use embedded SQL files with `goose` (github.com/pressly/goose). Supports `embed.FS` as migration source, `database/sql` compatible, lightweight (no external tool needed at runtime). Alternatively, simple manual migration in `db.go` via schema version tracking is acceptable for the small schema.

**Password hashing**: `golang.org/x/crypto/bcrypt` — standard, well-audited, cost factor 12 default.

---

## Decision 3: Web Server & Session Management

**Decision**: `net/http` stdlib + `github.com/go-chi/chi/v5` router + in-memory session store

**Rationale**: chi is 7KB with zero external dependencies. It provides clean middleware composition for auth protection without the bloat of Echo/Gin. For a 100-user admin tool with rare access, an in-memory session map (map[token]AdminSession) is perfectly sufficient. No persistence needed — sessions expire on restart.

**Alternatives considered**:
- `gorilla/sessions`: Full session management with pluggable stores. Overkill; adds securecookie dependency.
- Stdlib net/http only (no router): Would work, but chi's middleware grouping significantly reduces boilerplate for protected routes.
- Echo/Gin: Full frameworks unnecessary for 3-5 admin routes.

**Session details**:
- Generate 32-byte cryptographically random session token on login
- Store in httpOnly + SameSite=Strict cookie
- Server-side map[string]sessionData with expiry timestamps
- Validate on every protected route via chi middleware

**Templates**: `html/template` + `//go:embed templates/*.html` — zero runtime I/O, single binary.

**CSRF**: Embed a CSRF token (random, per-session) in every form as a hidden field. Validate on POST handlers. No library needed.

---

## Decision 4: Configuration Format

**Decision**: YAML file with `gopkg.in/yaml.v3`

**Rationale**: User explicitly specified YAML. `gopkg.in/yaml.v3` is the standard Go YAML library (indirect dep of many tools, highly stable).

**Schema**:
```yaml
radius:
  shared_secret: "change-me"
  port: 1812

database:
  path: ./radius.db

web:
  port: 8080
  session_secret: "32-char-random-string-here"

admins:
  - username: admin
    password: "plaintext-or-bcrypt-hash"
```

Admin passwords: accept plain text in config (bcrypt-hash on first read and rewrite, or just compare bcrypt at login). Simplest: store bcrypt-hashed in config. Document that admins use `htpasswd -bnBC 12 "" password | tr -d ':\n'` or provide a helper subcommand `radius-server hash-password`.

---

## Decision 5: Build Tooling

**Decision**: Makefile with targets: `build`, `test`, `lint`, `run`, `clean`

**Linter**: `golangci-lint` — standard Go linter aggregator. Install via `go install` or script.

**Pre-commit gate**: Makefile `pre-commit` target runs lint + test. Git pre-commit hook calls this target.

**Binary**: Single static binary via `go build -ldflags="-s -w" -trimpath ./cmd/server`. For Linux cross-compile: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...`.

---

## Resolved Unknowns

| Unknown | Resolution |
|---------|-----------|
| RADIUS library | layeh.com/radius (pure Go) |
| SQLite driver | glebarez/sqlite wrapping modernc.org/sqlite (pure Go) |
| CHAP implementation | Manual MD5 comparison (~10 lines) |
| Session management | In-memory map, chi middleware |
| Admin password storage | bcrypt-hashed in YAML config |
| MikroTik rate-limit attribute | Vendor 14988, Attr 8, string "Dk/Uk" format |
| Migration strategy | goose with embed.FS or inline schema in db.go |
