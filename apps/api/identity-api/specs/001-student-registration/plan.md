# Implementation Plan: Student Registration (CU-AUTH-01 / HU-01)

**Branch**: `001-student-registration` | **Date**: 2026-11-18 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-student-registration/spec.md` and the
plan-phase design input supplied with the `/speckit.plan` invocation.

## Summary

Register a visitor as a `STUDENT` through `POST /api/identity/auth/register`. The handler
stays thin: it decodes a JSON DTO and delegates to `RegisterUserUseCase.Execute`, which
normalizes and validates the email, enforces the password policy, hashes the password with
bcrypt, and atomically inserts a `users` row plus an empty `profiles` row inside one explicit
`*sql.Tx` built from `sqlc`-generated queries. A successful registration emits an in-process
`UserRegistered` event to downstream hooks via a best-effort, retrying publisher; failures of
that publisher never roll back the account. Duplicate normalized emails (case-insensitive) map
to a `409` conflict; validation failures map to `422`, malformed input to `400`, and an exceeded
per-visitor attempt budget to `429`. Configuration comes from environment variables; dependency
wiring is manual constructor injection in `cmd/api/main.go`.

The service is greenfield: no `go.work`, `go.mod`, `sqlc.yaml`, `goose` migrations, or
`ci-identity.yml` exists yet. Those are created as part of implementation; this plan is the
design contract for that work.

## Technical Context

**Language/Version**: Go 1.23+ (pin an exact version and `toolchain` directive in `go.mod` at
implementation time; no CGO, build with `CGO_ENABLED=0`).

**Primary Dependencies**: `github.com/go-chi/chi/v5`, `github.com/go-chi/cors`,
`github.com/go-chi/httprate` (approved stack; usage decided in [research.md](./research.md)),
`modernc.org/sqlite`, `golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5`
(service-level, not exercised by this feature), `github.com/pressly/goose/v3` and `sqlc` as
build/dev tools, `log/slog` from the standard library. `github.com/stretchr/testify` is
requested for tests and is a test-only dependency outside the enumerated stack — see the
Constitution Check and Complexity Tracking below for the required ADR follow-up.

**Storage**: Isolated SQLite 3 database via pure-Go `modernc.org/sqlite`. Default
`DB_PATH=./data/identity.db`. DSN enables WAL, `synchronous=NORMAL`, `foreign_keys=ON`,
`busy_timeout=5000`, and `cache_size=-64000`; `SetMaxOpenConns(1)` serializes v1 access.

**Testing**: `go test -race ./...`; table-driven unit tests with fakes for the use case and
domain; `net/http/httptest` handler tests; SQLite integration tests on `:memory:` with `goose`
migrations applied, covering atomic rollback and duplicate-email conflict. `golangci-lint run`
and a clean `sqlc generate` (no drift) are part of the verification gate.

**Target Platform**: Linux container/server, single-instance v1, `CGO_ENABLED=0`.

**Project Type**: Web service (microservice module inside a Go workspace).

**Performance Goals**: Registration completes in under 3 seconds end to end (SC-001). The
single-writer connection pool is acceptable at v1 scale; no separate throughput target is set.

**Constraints**: Pure standard-library domain layer; inward-only dependencies; consumer-defined
interfaces; manual dependency injection (no reflection DI); OpenAPI contract defined before
handlers; no raw SQL in handlers or use cases; no hardcoded secrets, keys, or paths; no new
third-party dependency without an ADR.

**Scale/Scope**: Single instance for v1. In-process event hooks and in-process per-visitor
lockout state; no external broker or shared rate-limit store. A durable outbox and a
distributed rate limiter are explicitly deferred (see [research.md](./research.md)).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Inward-only dependencies and contract-first HTTP | PASS | Layer mapping in [data-model.md](./data-model.md) matches the mandated boundaries. `internal/domain` imports only the standard library. Use case defines consumer-side ports and depends only on the domain. HTTP adapter owns `chi.NewRouter()`, `http.HandlerFunc`, JSON DTOs, and `respondError`. SQLite adapter wraps sqlc queries in an explicit `*sql.Tx`. Manual DI in `cmd/api/main.go`. OpenAPI contract is completed in Phase 1 before any handler work. |
| II | Approved stack and idiomatic Go | PASS | Stack is the approved set. Errors are wrapped with `fmt.Errorf("...: %w", err)` and matched with `errors.Is`/`errors.As`. |
| III | Phase-scoped Go skills | PASS | All five `.agents/skills/*/SKILL.md` were loaded for this plan phase. Spec and clarify phases remained technology-agnostic. |
| IV | Secure configuration and verified changes | PASS | All runtime settings come from env vars documented in `.env.example` (to be created in implementation). Passwords are bcrypt-hashed and never returned. Unit + integration tests are planned for every use case and repository adapter. Verification gate matches `ci-identity.yml` (to be created). |
| V | Focused changes and dependency discipline | **VIOLATION (tracked)** | `github.com/stretchr/testify` is requested for tests but is not in the enumerated approved stack. An explicit ADR is required before it is added to `go.mod`; the stdlib-only alternative is documented in Complexity Tracking. All other dependencies are approved. |

**Pre-design gate result**: PASS with one tracked violation (test-only `testify`, ADR required).
The violation does not block design because it affects test ergonomics only, not architecture or
runtime behavior. It must be resolved before implementation begins.

**Post-design re-check (Phase 1 complete)**: PASS with the same tracked violation. No new
violations were introduced by the research or contract design. The two design decisions that
could have created violations were resolved as follows: (a) UUID generation uses
`crypto/rand` from the standard library, so no UUID dependency is introduced; (b) the
per-visitor lockout is an in-process middleware rather than a new dependency, and `httprate`
remains an approved-stack option documented in research.

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
apps/api/identity-api/                      # Spec Kit project root (plan artifacts above)
services/
├── go.work                                 # Go workspace (module set created at implementation)
└── identity-service/                       # module lms-english/identity-service
    ├── go.mod
    ├── cmd/
    │   └── api/
    │       └── main.go                     # config, DB init, manual DI, graceful shutdown
    ├── internal/
    │   ├── domain/                         # stdlib-only entities, value objects, sentinel errors
    │   │   ├── user.go                     # User, Role enum, CEFRLevel value object
    │   │   ├── profile.go                  # Profile
    │   │   ├── event.go                    # UserRegisteredEvent
    │   │   ├── email.go                    # NormalizeEmail / ValidateEmail
    │   │   ├── password.go                 # ValidatePasswordPolicy
    │   │   └── errors.go                   # ErrInvalidEmail, ErrWeakPassword,
    │   │                                   # ErrEmailAlreadyExists (+ constitution sentinels)
    │   ├── usecase/                        # depends only on domain + stdlib
    │   │   ├── register_user.go            # RegisterUserUseCase.Execute
    │   │   └── ports.go                    # consumer-defined interfaces
    │   └── adapter/
    │       ├── http/                       # chi router, DTOs, respondError, rate-limit middleware
    │       │   ├── router.go
    │       │   ├── register_handler.go
    │       │   ├── respond_error.go
    │       │   └── visitor_limit.go
    │       ├── sqlite/                     # DB init + repository adapter over sqlc queries
    │       │   ├── db.go                   # SQLite connection initializer
    │       │   └── registration_store.go   # atomic RegisterUser transaction
    │       └── security/
    │           └── bcrypt_hasher.go        # BcryptHasher implements PasswordHasher
    ├── db/
    │   ├── migrations/
    │   │   └── 00001_create_users_and_profiles.sql   # goose Up/Down
    │   └── queries/
    │       └── users.sql                   # CreateUser :one, GetUserByEmail :one,
    │                                       # CreateProfile :exec
    ├── sqlc.yaml                           # engine: sqlite; schema: db/migrations;
    │                                       # queries: db/queries
    ├── contracts/
    │   └── identity.openapi.yaml           # module-local working copy of the contract
    ├── .env.example                        # APP_ENV, HTTP_PORT, DB_PATH (no real secrets)
    └── data/                               # runtime DB directory (gitignored)

shared/
└── contracts/
    └── identity.openapi.yaml               # canonical cross-service contract (mirror/publish
                                            # from specs/.../contracts/identity.openapi.yaml)
```

**Structure Decision**: The source tree is anchored at the Spec Kit project root
`apps/api/identity-api/`: a `services/go.work` workspace holds the single module
`lms-english/identity-service`. This follows the constitution's "inside the `services/` Go
workspace" rule. The canonical cross-service OpenAPI document is published to the monorepo
`shared/contracts/identity.openapi.yaml`; the module keeps a local copy under `contracts/` for
handler authors, and the design artifact under this feature's `contracts/` is the source of
truth for both. Because the repository is greenfield, all of these paths are created by the
implementation phase; the plan only defines them.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| `github.com/stretchr/testify` is a test-only dependency outside the approved stack (Principle V) | Requested plan-phase test strategy uses testify for expressive table-driven assertions and fake setup across domain, use case, SQLite, and HTTP tests | Stdlib `testing` with plain `if got != want { t.Errorf(...) }` is viable and requires no ADR. If an ADR is not approved before implementation, use stdlib-only tests instead of adding testify. |
| In-memory per-visitor lockout state (Principle IV / single-instance scope) | FR-011 requires a 30-minute lockout and suppression of remaining-time disclosure; a shared store is out of v1 scope | `github.com/go-chi/httprate` provides per-IP sliding windows but cannot express "5 attempts in 15 minutes → 30-minute penalty box" or suppress remaining-time disclosure cleanly; a distributed store would add an unapproved dependency. Documented as a v1 limitation with a deferred durable/distributed follow-up. |
