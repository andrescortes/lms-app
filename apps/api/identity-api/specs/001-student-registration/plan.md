# Implementation Plan: Student Registration (CU-AUTH-01 / HU-01)

**Branch**: `001-student-registration` | **Date**: 2026-11-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-student-registration/spec.md` and the
plan-phase design input supplied with the `/speckit.plan` invocation.

## Summary

Register a visitor as a `STUDENT` through `POST /api/identity/auth/register`. The handler
stays thin: it decodes a JSON DTO and delegates to `RegisterUserUseCase.Execute`, which trims
and case-normalizes the email, enforces the 8–72-character password policy, hashes the password
with bcrypt, and atomically inserts a `users` row plus an empty `profiles` row inside one explicit
`*sql.Tx` built from `sqlc`-generated queries. A successful registration records one logical
`UserRegistered` event in SQLite (durable outbox) and dispatches it to in-process downstream
hooks; delivery is retried and resumes after a service restart, and duplicate deliveries of the
same logical event do not cause duplicate downstream effects. Dispatch failures never roll back
the account. Duplicate normalized emails (case-insensitive) map to a `409` conflict; validation
failures (email format, password length/uppercase/digit) map to `422` and name the specific rule,
malformed input to `400`, and an exceeded per-visitor attempt budget to `429` (sixth attempt in a
15-minute window; other rejection outcomes use their own standard codes/messages). Configuration
comes from environment variables; dependency wiring is manual constructor injection in
`cmd/api/main.go`.

The service is greenfield: no `go.work`, `go.mod`, `sqlc.yaml`, `goose` migrations, or
`ci-identity.yml` exists yet. Those are created as part of implementation; this plan is the
design contract for that work.

## Technical Context

**Language/Version**: Go 1.23+ (pin an exact version and `toolchain` directive in `go.mod` at
implementation time; no CGO, build with `CGO_ENABLED=0`).

**Primary Dependencies**: `github.com/go-chi/chi/v5`, `github.com/go-chi/cors`,
`github.com/go-chi/httprate` (approved stack; usage decided in [research.md](./research.md)),
`modernc.org/sqlite`, `golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5`
(service-level; intentionally **not** pinned in `go.mod` until the deferred authentication
feature lands, so the unused module is not carried — see T058/T067), `github.com/pressly/goose/v3`
and `sqlc` as build/dev tools, and `log/slog` from the standard library. Tests use Go
`testing`/`httptest` only; `github.com/stretchr/testify` is excluded, so no ADR is required.

**Storage**: Isolated SQLite 3 database via pure-Go `modernc.org/sqlite`. `DB_PATH` is required
and supplied by the environment; `./data/identity.db` is a local example, not a hardcoded
default. DSN enables WAL, `synchronous=NORMAL`, `foreign_keys=ON`,
`busy_timeout=5000`, and `cache_size=-64000`; `SetMaxOpenConns(1)` serializes v1 access. The
`registration_events` outbox table stores one logical `UserRegistered` event per registration
for restart-safe delivery.

**Testing**: `go test -race ./...`; table-driven unit tests with fakes for the use case and
domain; `net/http/httptest` handler tests; SQLite integration tests on isolated temp-file
databases (the configured WAL/page-cache PRAGMAs require a file-backed database) with `goose`
migrations applied, covering atomic rollback, duplicate-email conflict, outbox record creation,
restart-safe redelivery, and duplicate-event idempotency at consumers. `golangci-lint run` and a
clean `sqlc generate` (no drift) are part of the verification gate.

**Target Platform**: Linux container/server, single-instance v1, `CGO_ENABLED=0`.

**Project Type**: Web service (microservice module inside a Go workspace).

**Performance Goals**: Registration completes in under 3 seconds end to end (SC-001). The
single-writer connection pool is acceptable at v1 scale; no separate throughput target is set.

**Constraints**: Pure standard-library domain layer; inward-only dependencies; consumer-defined
interfaces; manual dependency injection (no reflection DI); OpenAPI contract defined before
handlers; no raw SQL in handlers or use cases; no hardcoded secrets, keys, or paths; no new
third-party dependency without an ADR.

**Scale/Scope**: Single instance for v1. Durable outbox table plus in-process hooks for event
delivery (no external broker); in-process per-visitor lockout state; no shared rate-limit store.
A distributed rate limiter and an external broker remain explicitly deferred
(see [research.md](./research.md)).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Inward-only dependencies and contract-first HTTP | PARTIAL — service-wide mandates deferred | This feature implements the inward-only boundaries and contract-first HTTP (stdlib-only domain, consumer ports, chi/http.HandlerFunc/DTOs/respondError, sqlc-backed adapter, manual DI). The constitution's service-wide MUSTs for JWT/RBAC middleware in `internal/adapter/http` and JWT signing/validation in `internal/adapter/security` are not delivered by this public-registration feature and are tracked in tasks.md Phase 7 (T058). This is a scope deferral, not a reinterpretation. |
| II | Approved stack and idiomatic Go | PASS | Stack is the approved set. Errors are wrapped with `fmt.Errorf("...: %w", err)` and matched with `errors.Is`/`errors.As`. |
| III | Phase-scoped Go skills | PASS | All five `.agents/skills/*/SKILL.md` were loaded for this plan phase. Spec and clarify phases remained technology-agnostic. |
| IV | Secure configuration and verified changes | PASS | All runtime settings come from env vars documented in `.env.example` (to be created in implementation). Passwords are bcrypt-hashed and never returned. Unit + integration tests are planned for every use case and repository adapter. Verification gate matches `ci-identity.yml` (to be created). |
| V | Focused changes and dependency discipline | PASS | Only approved dependencies are used; tests use Go stdlib `testing`/`httptest`, so testify and its ADR are not needed. |

**Pre-design gate result**: PASS. Only approved dependencies are used; tests use Go stdlib
`testing`/`httptest`, so no test-dependency ADR is needed.

**Post-design re-check (Phase 1 complete; clarifications applied)**: PASS. No new violations
were introduced by the research, contract design, or the 2026-11-18 clarifications. The design
decisions that could have created violations were resolved as follows:
(a) UUID generation uses `crypto/rand` from the standard library, so no UUID dependency is
introduced; (b) the per-visitor lockout is an in-process middleware rather than a new dependency,
and `httprate` remains an approved-stack option documented in research; (c) restart-safe event
delivery uses a SQLite outbox table and in-process hooks only — no external broker or new
dependency — so Principle V remains unaffected.

## Project Structure

### Documentation (this feature)

```text
specs/001-student-registration/
├── spec.md              # Feature specification (done)
├── plan.md              # This file (/speckit.plan output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── identity.openapi.yaml
├── checklists/
│   └── requirements.md  # Spec quality checklist (done)
└── tasks.md             # Phase 2 output (/speckit.tasks - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
apps/api/identity-api/                      # Spec Kit project root AND Go module root
├── go.mod                                  # module lms-english/identity-service
├── cmd/
│   └── api/
│       └── main.go                         # config, DB init, manual DI, graceful shutdown
├── internal/
│   ├── domain/                             # stdlib-only entities, value objects, sentinel errors
│   │   ├── user.go                         # User, Role enum, CEFRLevel value object
│   │   ├── profile.go                      # Profile
│   │   ├── event.go                        # UserRegisteredEvent
│   │   ├── email.go                        # NormalizeEmail / ValidateEmail
│   │   ├── password.go                     # ValidatePasswordPolicy
│   │   └── errors.go                       # ErrInvalidEmail, ErrWeakPassword,
│   │                                       # ErrEmailAlreadyExists (+ constitution sentinels)
│   ├── usecase/                            # depends only on domain + stdlib
│   │   ├── register_user.go                # RegisterUserUseCase.Execute
│   │   └── ports.go                        # consumer-defined interfaces
│   └── adapter/
│       ├── http/                           # chi router, DTOs, respondError, rate-limit middleware
│       │   ├── router.go
│       │   ├── register_handler.go
│       │   ├── respond_error.go
│       │   └── visitor_limit.go
│       ├── sqlite/                         # DB init + repository adapter over sqlc queries
│       │   ├── db.go                       # SQLite connection initializer
│       │   └── registration_store.go       # atomic RegisterUser transaction
│       ├── security/
│       │   └── bcrypt_hasher.go            # BcryptHasher implements PasswordHasher
│       └── event/                          # in-process hooks + outbox retry worker
│           └── publisher.go                # DomainEventPublisher; restart-safe redelivery
├── db/
│   ├── migrations/
│   │   ├── 00001_create_users_and_profiles.sql   # goose Up/Down
│   │   └── 00002_create_registration_events.sql  # outbox (human-reviewed)
│   └── queries/
│       ├── users.sql                       # CreateUser :one, GetUserByEmail :one,
│       │                                   # CreateProfile :exec
│       └── events.sql                      # CreateEvent, ListPending, MarkDelivered
├── sqlc.yaml                               # engine: sqlite; schema: db/migrations;
│                                           # queries: db/queries
├── contracts/
│   └── identity.openapi.yaml               # module-local working copy of the contract
├── .env.example                            # APP_ENV, HTTP_PORT, DB_PATH (no real secrets)
└── data/                                   # runtime DB directory (gitignored)

services/
└── go.work                                 # workspace referencing root module via `use ..`

shared/
└── contracts/
    └── identity.openapi.yaml               # canonical cross-service contract (mirror/publish
                                            # from specs/.../contracts/identity.openapi.yaml)
```

**Structure Decision**: The Go module `lms-english/identity-service` is rooted directly at the
Spec Kit project root `apps/api/identity-api/`; `services/go.work` includes that root module via
`use ..`, satisfying the constitution's workspace requirement without a nested service directory.
Commands set `GOWORK` to the workspace's absolute path. The canonical cross-service OpenAPI
document is published to the monorepo `shared/contracts/identity.openapi.yaml`; the module keeps a
local copy under `contracts/` for handler authors, and the design artifact under this feature's
`contracts/` is the source of truth for both. Because the repository is greenfield, all of these
paths are created by the implementation phase; the plan only defines them.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified

| Violation / Deferred Item | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Deferred service-wide mandates (Principle I) | Registration is unauthenticated; JWT/RBAC and token issuance belong to a later auth feature | Adding token endpoints here would exceed CU-AUTH-01/HU-01 scope; tracked explicitly in tasks.md T058 instead of silently omitted |
| In-memory per-visitor lockout state (Principle IV / single-instance scope) | FR-011 requires a 30-minute lockout (first five attempts allowed, sixth triggers) and suppression of remaining-time disclosure; a shared store is out of v1 scope | `github.com/go-chi/httprate` provides per-IP sliding windows but cannot express "first five allowed, sixth → 30-minute penalty box" or suppress remaining-time disclosure cleanly; a distributed store would add an unapproved dependency. Documented as a v1 limitation with a deferred durable/distributed follow-up. |
