<!--
Sync Impact Report (temporary; remove before committing)
Version change: unratified scaffold → 1.0.0 (initial adoption)
Modified principles: No prior ratified principles. The five scaffold slots now define:
- I. Inward-Only Dependencies and Contract-First HTTP
- II. Approved Stack and Idiomatic Go
- III. Phase-Scoped Go Skills
- IV. Secure Configuration and Verified Changes
- V. Focused Changes and Dependency Discipline
Added sections: Approved Technology Stack; Spec Kit Workflow and Verification;
  Governance (amendment, versioning, and compliance rules).
Removed sections: None.
Follow-up TODOs: None.
-->

# Identity Service (`identity-service`) Constitution

## Core Principles

### I. Inward-Only Dependencies and Contract-First HTTP

The `identity-service` module MUST follow inward-only dependencies in the `services/` Go
workspace (`services/go.work`). Business rules MUST remain independent of transport,
persistence, and authentication libraries.

- `internal/domain` MUST contain pure domain entities, value objects, and sentinel errors,
  including `ErrInvalidCredentials`, `ErrForbidden`, and `ErrNotFound`. It MUST import only
  the Go standard library and MUST NOT import `net/http`, `database/sql`, or JWT packages.
- `internal/usecase` MUST contain application logic, accept and propagate `context.Context`,
  and define the output ports `UserRepository`, `PasswordVerifier`, `TokenProvider`, and
  `RefreshTokenRepository`. It MUST depend only on `internal/domain` and the standard library.
- `internal/adapter/http` MUST own `chi.NewRouter()` routing, `http.HandlerFunc` handlers,
  JSON DTOs, JWT authentication and role-based access control (RBAC) middleware, and centralized
  error-to-response mapping via `respondError(w, r, err)`. It MUST delegate business rules to
  use cases and use injected interfaces for security operations rather than import sibling adapters.
- `internal/adapter/sqlite` MUST implement repository ports using `sqlc`-generated queries
  over `*sql.DB`. `internal/adapter/security` MUST implement JWT signing and validation with
  `github.com/golang-jwt/jwt/v5` and password hashing and verification with `bcrypt`.
- `cmd/api/main.go` MUST load configuration, initialize SQLite, wire dependencies explicitly,
  and shut down HTTP gracefully. Reflection-based dependency injection containers MUST NOT be used.
- OpenAPI contracts in `shared/contracts/` MUST define new or changed HTTP behavior before
  the corresponding handlers are implemented. Handlers MUST conform to those contracts.

### II. Approved Stack and Idiomatic Go

The service MUST use the Approved Technology Stack below. Go code MUST follow standard library
conventions, preserve error causes with `fmt.Errorf("...: %w", err)` when adding context, and
use `errors.Is` for sentinel matching and `errors.As` for typed errors. Handlers and use cases
MUST NOT embed or execute raw SQL; service queries belong in versioned `sqlc` inputs.

### III. Phase-Scoped Go Skills

Before technical design, task breakdown, code generation, or technical analysis, OpenCode MUST
find, load, and apply all Go skills at `.agents/skills/*/SKILL.md` in the Identity project.
Use them by phase as defined below to keep functional specifications free of implementation choices
and apply Go, Clean Architecture, `chi`, `sqlc`, error handling, and testing conventions in technical work.

### IV. Secure Configuration and Verified Changes

The service MUST load all runtime configuration from environment variables documented in
`.env.example`. JWT secrets, keys, and runtime file paths MUST NOT be hardcoded; example files
MUST NOT contain real credentials. Every use case MUST have automated unit tests, and every
SQLite repository adapter MUST have automated integration tests against SQLite. The verification
checks below MUST pass before a service change is complete.

### V. Focused Changes and Dependency Discipline

Agents MUST make concise, targeted edits limited to the active task and MUST NOT rewrite unrelated
files. A third-party dependency outside the approved stack MUST NOT be introduced without an
explicit, approved Architectural Decision Record (ADR) documenting the rationale and scope.
Replacing a mandated stack component also requires a constitutional amendment.

## Approved Technology Stack

- **Language and HTTP:** Go with `github.com/go-chi/chi/v5`, `github.com/go-chi/cors`, and
  `github.com/go-chi/httprate`.
- **Persistence:** An isolated SQLite 3 database using the pure-Go `modernc.org/sqlite` driver
  (no CGO dependency). Configure the database path through the environment; `/data/identity.db`
  and `./data/identity.db` are deployment and local examples, not hardcoded defaults. SQLite
  MUST use `PRAGMA journal_mode=WAL;` and enforce foreign keys on every connection.
- **Queries and migrations:** `sqlc.yaml` MUST specify `engine: "sqlite"` and use queries in
  `db/queries/*.sql`. Deterministic `goose` migrations MUST reside in `db/migrations/`.
- **Authentication:** Use `github.com/golang-jwt/jwt/v5` for short-lived access tokens and
  rotating refresh tokens. Each successful refresh MUST revoke the previous refresh token by
  `jti` in `identity.db`; revoked refresh tokens MUST be rejected.

## Spec Kit Workflow and Verification

- **`/speckit.specify` and `/speckit.clarify`:** MUST NOT apply Go implementation skills or stack
  constraints. Keep `spec.md` functional and technology-agnostic.
- **`/speckit.plan` and `/speckit.tasks`:** MUST use the discovered Go skills to map requirements
  to Clean Architecture layers, OpenAPI contracts, `sqlc` queries, and `goose` migrations.
- **`/speckit.implement` and `/speckit.analyze`:** MUST enforce the discovered Go skills, including
  idiomatic concurrency, cancellation and context propagation, and table-driven tests.
- **Verification gate:** Match `ci-identity.yml`. Run `sqlc generate` and verify that it produces
  no uncommitted drift in generated files. `golangci-lint run` and `go test -race ./...` MUST pass
  for the Identity module in the Go workspace.

## Governance

This constitution governs Identity service work. Plans, task lists, and pull request reviews MUST
verify compliance with the rules applicable to their phase. Service pull requests MUST document
verification results and reference any required ADR before merge.

An amendment MUST be proposed in a pull request that explains the rationale, identifies affected
principles and workflows, and includes a migration plan when existing work is affected. Project
maintainers MUST approve the amendment before it takes effect. Update the constitution version and
last-amended date on adoption; preserve the original ratification date.

Version amendments use semantic versioning: MAJOR for incompatible removals or redefinitions of
principles or governance rules; MINOR for new principles or sections or materially expanded
requirements that do not invalidate existing governance; PATCH for non-semantic clarifications
and corrections.

**Version**: 1.0.0 | **Ratified**: 2026-11-18 | **Last Amended**: 2026-11-18
