# Quickstart: Student Registration Validation

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) |
**Contract**: [contracts/identity.openapi.yaml](./contracts/identity.openapi.yaml) |
**Data model**: [data-model.md](./data-model.md)

This guide validates the Student Registration implementation end to end. It is a run book, not
an implementation guide. All commands are run from the Spec Kit project root
`apps/api/identity-api/`.

## Prerequisites

- Go 1.23+ with `CGO_ENABLED=0` supported.
- `sqlc` and `goose` binaries available on `PATH`.
- `golangci-lint` available on `PATH` for the verification gate.
- Ports: `HTTP_PORT` (default `8081`) free.

## 1. Module and workspace layout

The Go module `lms-english/identity-service` is rooted at the project root
`apps/api/identity-api/` (where `go.mod` lives). `services/go.work` includes it via `use ..`.

```bash
# from apps/api/identity-api/
export GOWORK="$PWD/services/go.work"   # workspace lives below the module
go list ./...
```

Expected: `go list ./...` resolves the module; `services/go.work` reports `use ..`.

## 2. Generate and migrate

```bash
# from apps/api/identity-api/
export GOWORK="$PWD/services/go.work"
sqlc generate
go run github.com/pressly/goose/v3/cmd/goose -dir db/migrations sqlite3 ./data/identity.db up
```

Expected:
- `sqlc generate` succeeds, then running it a second time produces **no diff** in generated
  files (verification gate).
- The `users` and `profiles` tables exist with the constraints described in
  [data-model.md](./data-model.md).
- Verify pragmas on the running database: `journal_mode` is `wal` and `foreign_keys` is `1`.

## 3. Configure the service

Copy `.env.example` to `.env` (documented keys: `APP_ENV`, `HTTP_PORT`, `DB_PATH`). No secrets
or absolute file paths are hardcoded in source. For local validation:

```bash
export APP_ENV=local
export HTTP_PORT=8081
export DB_PATH=./data/identity.db
```

## 4. Run the automated checks

```bash
# from apps/api/identity-api/
export GOWORK="$PWD/services/go.work"
go test ./...            # add -race in a CGO-capable environment
golangci-lint run
```

Expected: both pass. Integration tests use isolated temp-file SQLite databases with migrations
applied and cover atomic rollback and duplicate-email conflict; handler tests use
`net/http/httptest`.

## 5. Start the service and validate the contract

```bash
# from apps/api/identity-api/
export GOWORK="$PWD/services/go.work"
go run ./cmd/api
```

Expected: the server binds to `HTTP_PORT` and shuts down gracefully on SIGTERM/SIGINT.

### Scenario A — Successful registration (201)

```bash
curl -sS -i -X POST http://localhost:8081/api/identity/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"Student@Example.com","password":"Learn123"}'
```

Expected: `201 Created`; JSON body contains only `{"userId":"<uuid>"}`; the stored email is
`student@example.com`, role is `STUDENT`, the profile exists with all optional fields unset, and
one `UserRegistered` event is published. Omitting `displayName` stores `student` as the display
name.

### Scenario B — Duplicate email, case-insensitive (409)

```bash
curl -sS -i -X POST http://localhost:8081/api/identity/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"student@example.com","password":"Learn123"}'
```

Expected: `409` with `code: email_already_exists` in the standard
`{"code","message","timestamp","path"}` envelope; no existing-account details are disclosed.

### Scenario C — Invalid email and weak password (422)

```bash
curl -sS -i -X POST http://localhost:8081/api/identity/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"not-an-email","password":"weakpass"}'
```

Expected: `422` with `code: validation_error`; `message` names the violated rule; no account is
created and no event is emitted.

### Scenario D — Malformed request (400)

```bash
curl -sS -i -X POST http://localhost:8081/api/identity/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com",'
```

Expected: `400` with `code: invalid_request`.

### Scenario E — Per-visitor lockout (429)

Submit five registration attempts from the same client within 15 minutes (the first five are
allowed, including invalid attempts), then submit a sixth.

Expected: the sixth returns `429` with `code: rate_limited`, a fixed message, **no `Retry-After`
header**, and no remaining-time disclosure. A different client IP is unaffected. After 30 minutes
the visitor can register again.

## 6. Contract conformance

Validate request and response shapes against
[contracts/identity.openapi.yaml](./contracts/identity.openapi.yaml). Every non-2xx response MUST
use the single `ErrorResponse` envelope produced by `respondError`; no endpoint may invent a
different error shape.

## Definition of Done for this feature

- `sqlc generate` produces no uncommitted drift.
- `golangci-lint run` passes.
- `go test -race ./...` passes (in a CGO-capable environment).
- All five curl scenarios match the expected outcomes in the contract.
- `ci-identity.yml` encodes the verification gate and passes in CI.
