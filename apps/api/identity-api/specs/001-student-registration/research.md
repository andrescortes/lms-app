# Research: Student Registration (CU-AUTH-01 / HU-01)

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

**Purpose**: Resolve every design decision and technology question raised by the feature spec
and the plan-phase input before Phase 1 design. Each decision is recorded as Decision /
Rationale / Alternatives considered.

## R1 — Greenfield anchoring of module and workspace

- **Decision**: Create `services/go.work` and the `lms-english/identity-service` module under
  the Spec Kit project root `apps/api/identity-api/`. Publish the canonical OpenAPI document to
  the monorepo `shared/contracts/identity.openapi.yaml`, with a module-local copy at
  `services/identity-service/contracts/identity.openapi.yaml`.
- **Rationale**: The constitution mandates a `services/` Go workspace and OpenAPI contracts in
  `shared/contracts/`. Anchoring at the project root keeps all feature artifacts and source in
  one reviewable unit; the monorepo path satisfies cross-service contract discovery. No such
  files exist yet, so implementation creates them.
- **Alternatives considered**: Placing `services/` at the monorepo root would scatter the Go
  workspace away from the Spec Kit project that owns the spec; rejected. Keeping only one copy
  of the contract risks handler drift; rejected in favor of an explicit publish step.

## R2 — UUID generation without a new dependency

- **Decision**: Implement `IDGenerator` using `crypto/rand` and standard-library formatting to
  produce RFC 4122 version 4 UUIDs (128-bit random, correct version/variant bits). Do not add a
  UUID package.
- **Rationale**: Principle V forbids unapproved third-party dependencies. `crypto/rand` satisfies
  the security skill's requirement for unpredictable identifiers and removes an ADR obligation.
  Version 4 is sufficient because IDs need only be unique and opaque, not time-sortable.
- **Alternatives considered**: `github.com/google/uuid` (v4/v7 convenience) would add an
  unapproved dependency and an ADR; UUIDv7 (time-sortable) has no consumer requirement in this
  feature and is deferred.

## R3 — Per-visitor rate limiting and 30-minute lockout

- **Decision**: Implement a small in-process middleware in `internal/adapter/http`
  (`visitor_limit.go`) that keys a bounded, mutex-protected in-memory map by visitor identity.
  It records attempt timestamps, applies the rule "more than 5 attempts within 15 minutes →
  reject all attempts for 30 minutes", and returns `429` with a fixed body that does **not**
  include `Retry-After` or remaining time. Expired entries are pruned opportunistically and the
  map is capacity-bounded.
- **Rationale**: FR-011 and the clarification session require a penalty-box lockout, not merely
  a sliding-window limit: 5 attempts / 15 minutes triggers a fixed 30-minute lockout, and the
  response must not disclose remaining time. A purpose-built middleware models this exactly and
  keeps the policy testable in isolation.
- **Alternatives considered**: `github.com/go-chi/httprate` is in the approved stack and can
  limit by IP with a sliding window, but it cannot express a fixed-length penalty box after a
  threshold and naturally encourages disclosing rate-limit state; it remains approved and may be
  used for a simpler generic limit later. A shared/Redis-style store would give cross-instance
  consistency but introduces an unapproved dependency and is out of v1 scope.

## R4 — Deriving the visitor identity for the lockout

- **Decision**: Use the TCP peer host from `r.RemoteAddr` (via `net.SplitHostPort`) as the v1
  visitor key. Do not trust `X-Forwarded-For` or `X-Real-IP` by default. If a trusted reverse
  proxy is introduced, enable chi's `middleware.RealIP` only behind an explicit trusted-proxy
  allowlist.
- **Rationale**: The security skill flags client-supplied forwarding headers as trivially
  forgeable; keying a lockout on them would let an attacker bypass the limit or lock out other
  visitors. The peer address is server-controlled.
- **Alternatives considered**: Keying on email would let an attacker lock out a known victim and
  would leak account existence through lockout behavior; rejected. Cookie/session identity does
  not exist for unauthenticated registration; rejected.

## R5 — Error payload shape and status mapping

- **Decision**: Keep the user-specified error envelope
  `{"code","message","timestamp","path"}` with `Content-Type: application/json`, produced only
  by `respondError(w, r, err)`. Mapping: `ErrInvalidEmail` / `ErrWeakPassword` → `422`;
  `ErrEmailAlreadyExists` → `409`; malformed JSON / unknown fields / missing required fields →
  `400`; rate-limited → `429`; unexpected errors → `500`. Timestamps use
  `time.Now().UTC().Format(time.RFC3339Nano)` and `path` uses `r.URL.Path`. A small typed error
  (for example `validationError{field, reason}`) carries per-rule detail; sentinel errors are
  matched with `errors.Is`.
- **Rationale**: The plan input specifies this exact payload, so the OpenAPI contract and tests
  conform to it. A stable machine-readable `code` preserves the key benefit of RFC 9457 without
  changing the agreed wire format. `422` is used for well-formed-but-invalid input and `400` for
  malformed transport input, matching REST API conventions.
- **Alternatives considered**: RFC 9457 `application/problem+json` (recommended by the REST
  skill) was considered but would change the explicitly requested contract; document the
  deviation here rather than silently switching formats. Returning `400` for validation failures
  was rejected because it conflates malformed and semantically invalid input.

## R6 — Transaction boundary and sqlc integration

- **Decision**: Define one consumer-side port `UserRegistrationStore.Register(ctx, user, profile)`
  implemented in `internal/adapter/sqlite`. The adapter opens an explicit `*sql.Tx` via
  `BeginTx(ctx, nil)`, constructs a sqlc `Queries` bound with `WithTx(tx)`, calls
  `CreateUser` then `CreateProfile`, and commits. Any error rolls back via `defer tx.Rollback()`.
  Add compile-time interface assertions (`var _ usecase.UserRegistrationStore = (*Store)(nil)`).
- **Rationale**: FR-006/FR-008 require the `users` and `profiles` inserts to be atomic so no
  half-created account survives a failure or concurrent duplicate. `WithTx` is the idiomatic
  sqlc mechanism and keeps generated SQL out of handlers and use cases. The database skill
  requires context-aware calls and explicit transaction handling.
- **Alternatives considered**: A generic `TxManager` plus separate `UserRepository` and
  `ProfileRepository` ports was considered; it adds interfaces without improving this single
  aggregate write, so the narrower store port is preferred (design-patterns skill: smallest
  pattern that works). Letting each repository open its own transaction was rejected as it
  breaks atomicity.

## R7 — Detecting duplicate email and mapping to the domain error

- **Decision**: Rely on the `UNIQUE` constraint on `users.email`. In the adapter, detect a
  uniqueness violation from the `modernc.org/sqlite` error (driver error code / message for
  constraint `users.email`) and translate it to `domain.ErrEmailAlreadyExists` by wrapping with
  `%w`, so `errors.Is` works at the HTTP boundary. Do not pre-check with `GetUserByEmail` as the
  authority — the unique index is the concurrency-safe arbiter; `GetUserByEmail` is generated for
  supporting flows/tests only.
- **Rationale**: A check-then-insert race can create duplicates under concurrent submissions;
  the database constraint is authoritative (FR-008). Returning a sentinel keeps the HTTP layer
  free of driver-specific error handling.
- **Alternatives considered**: A `SELECT` before insert was rejected as the sole guard because
  of the TOCTOU race. Regex-matching driver error strings was accepted only as the pragmatic
  `modernc.org/sqlite` mechanism, with a comment documenting the constraint name and a test
  locking in the behavior; verify the exact driver error shape during implementation.

## R8 — sqlc reading goose migrations

- **Decision**: Configure `sqlc.yaml` with `engine: "sqlite"`, `schema: "db/migrations"`,
  `queries: "db/queries"`, and generated output in `internal/adapter/sqlite/sqlc` (or an
  equivalent package). Keep the goose `-- +goose Up` / `-- +goose Down` directives in
  `00001_create_users_and_profiles.sql`. Verify at implementation time that `sqlc generate`
  produces the intended tables (i.e., the Down section is not applied to the model).
- **Rationale**: The user-specified configuration is honored, and sqlc understands goose-style
  migration files. Verification closes the one risk: if a tool version applies the `Down` DDL,
  the generated model would lose the tables.
- **Alternatives considered**: Maintaining a separate Up-only schema snapshot for sqlc would
  avoid the risk but creates two schema sources that can drift; adopt it only if verification
  shows the Down statements are applied. Generating SQL by hand is forbidden by the constitution.

## R9 — Best-effort event delivery with retries

- **Decision**: `DomainEventPublisher` is an in-process hook dispatcher (no external broker).
  Publishing invokes registered hooks synchronously inside a retry helper that applies bounded
  exponential backoff, checks `ctx.Err()` between attempts, and logs a structured warning on
  exhaustion. A publisher error is never propagated through `RegisterUserUseCase.Execute`, so a
  delivery failure cannot roll back or fail a registration (FR-009, clarification Q1).
- **Rationale**: The clarification answer chose "registration succeeds regardless; notification
  retried until it lands". An in-process design matches the user's "synchronous/hook dispatcher
  for v1, no external broker" requirement while honoring the retry intent. Context checks
  follow the design-patterns skill's retry guidance.
- **Alternatives considered**: A durable outbox table plus worker would survive restarts but
  adds a table, queries, and scope beyond the specified `users`/`profiles` schema; deferred with
  an ADR for a future iteration. Synchronous fire-and-forget without retries was rejected by the
  clarification answer. Fire-and-forget goroutines were rejected as untracked work that races
  process shutdown.

## R10 — SQLite pragmas and single-writer pool

- **Decision**: Use the exact DSN supplied in the plan input:
  `_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=cache_size(-64000)`.
  Set `SetMaxOpenConns(1)` on the `*sql.DB`; verify with `PRAGMA journal_mode` and
  `PRAGMA foreign_keys` on startup. Run migrations with goose before serving traffic.
- **Rationale**: WAL plus foreign keys are mandated by the constitution; the remaining pragmas
  improve durability/locking behavior for a single-writer embedded database. One open connection
  makes writes deterministic for v1 and satisfies "isolated SQLite database". Startup
  verification converts a misconfigured database into a fail-fast error.
- **Alternatives considered**: A larger pool would allow concurrent reads but risks SQLite
  write-lock contention and is unnecessary at v1 scale; revisit under load. File path comes from
  `DB_PATH` only — the documented `/data/identity.db` and `./data/identity.db` are examples, not
  hardcoded defaults, per the secrets policy.

## R11 — Unset optional fields at registration

- **Decision**: At registration, leave `CEFRLevel` unset (nullable `TEXT`, `NULL`) and create a
  `Profile` with `native_language`, `goals`, `timezone`, and `avatar_url` all `NULL`. The `Role`
  enum starts at 1 with `RoleUnknown = 0` so the zero value is invalid; registration always
  assigns `STUDENT`.
- **Rationale**: FR-006 and the spec Assumptions require an empty profile at registration. The
  enum convention follows the design-patterns skill (zero value = unset/invalid), preventing an
  accidental zero-value role from passing validation.
- **Alternatives considered**: Empty strings instead of `NULL` were rejected because they are
  indistinguishable from a deliberately blank user value and weaken the "unset" semantics.

## R12 — Display name default

- **Decision**: The request DTO accepts optional `displayName`; when omitted or blank, derive it
  from the local part of the normalized email (text before `@`). Store the result on `User`.
- **Rationale**: This is explicitly specified in the plan input and gives every account a
  non-empty display value without extra user input.
- **Alternatives considered**: Leaving display name `NULL` was rejected because the plan input
  calls for a default; random handles were rejected as surprising to the visitor.

## R13 — Test strategy and the testify ADR

- **Decision**: Write table-driven tests at every layer: domain validation (pure), use case
  (fakes for store/hasher/id generator/clock/publisher), SQLite adapter on `:memory:` with goose
  migrations and `foreign_keys=ON`, and HTTP handlers with `net/http/httptest`. Run
  `go test -race ./...`. If the ADR for `github.com/stretchr/testify` is approved, use it for
  assertions; otherwise use stdlib `testing` assertions.
- **Rationale**: The constitution requires tests for every use case and repository adapter, and
  the verification gate requires the race detector. In-memory SQLite with migrations applied
  exercises the real constraint and transaction behavior without external services.
- **Alternatives considered**: Mocking the database entirely was rejected because it would not
  verify the UNIQUE violation mapping or rollback; a containerized database was rejected as
  unnecessary for SQLite and slower in CI.

## R14 — Verification gate and missing CI file

- **Decision**: Treat the verification gate as: `sqlc generate` produces no uncommitted drift,
  `golangci-lint run` passes, and `go test -race ./...` passes. Implementation must also add the
  referenced `ci-identity.yml` workflow (absent in the greenfield repo) encoding these checks.
- **Rationale**: The constitution explicitly matches the gate to `ci-identity.yml`. A plan cannot
  pass a gate in a file that does not exist; recording the gap keeps implementation honest.
- **Alternatives considered**: Skipping the CI file was rejected because the constitution names
  it as the verification reference; deferring it silently would leave the gate unenforceable.
