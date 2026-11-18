---
description: "Implementation tasks for Student Registration (CU-AUTH-01 / HU-01)"
---

# Tasks: Student Registration (CU-AUTH-01 / HU-01)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md),
[data-model.md](data-model.md), [OpenAPI contract](contracts/identity.openapi.yaml), and
[quickstart.md](quickstart.md).

**Status**: Task list prepared; implementation is blocked on T001–T002, which reconcile material
contradictions in the existing design documents. Do not treat the earlier plan's PASS as proof
that these conflicts are resolved.

**Paths**: Unless prefixed `repo:`, paths are relative to the current microservice root,
`apps/api/identity-api/`. `repo:` paths are relative to the Git repository root. The Go module
belongs directly in the microservice root, not in `services/identity-service/`. A workspace at
`services/go.work` can reference the root module with `use ..`; because it is below the module,
commands must explicitly set `GOWORK` to its absolute path. The root-level GitHub workflow is
`repo:.github/workflows/ci-identity.yml`; the canonical shared contract is
`repo:shared/contracts/identity.openapi.yaml`.

**Dependencies and tests**: Use only the approved stack and Go's `testing`/`httptest` with local
fakes; do not add `testify`, so its proposed ADR is unnecessary. Migration SQL is authored and
reviewed by a database owner, per the loaded database skill. Table-driven unit, integration,
contract, and race tests are required. `CGO_ENABLED=0` applies to the service build; run the race
check separately with the platform's race-detector prerequisites, including CGO where required.

## Format: `[ID] [P?] [Story] Description`

- `[P]`: independent, different-file task that may run in parallel with other tasks in its group
  after its stated prerequisites have completed.
- `[US1]`–`[US4]`: the user story from `spec.md`; setup, foundation, and polish are unlabeled.
- Story phases contain the requested inward-core, adapter/delivery, and entrypoint work in that
  order; the final cross-cutting phase completes the requested CI and verification stage.

## Phase 1: Setup & Greenfield Scaffolding

**Purpose**: Resolve design blockers, initialize the root module, and prepare configuration and
the contract before handler implementation.

- [ ] T001 Reconcile and obtain owner decisions in `specs/001-student-registration/spec.md`, `specs/001-student-registration/data-model.md`, and `specs/001-student-registration/contracts/identity.openapi.yaml` for whitespace rejection versus trimming, the lockout threshold and visitor identity (including shared IPs), and the password policy's lack of a maximum versus bcrypt's 72-byte input limit; define explicit acceptance tests and never silently truncate passwords.
- [ ] T002 Reconcile `specs/001-student-registration/spec.md`, `specs/001-student-registration/plan.md`, `specs/001-student-registration/research.md`, and `specs/001-student-registration/data-model.md` for FR-009 retry-until-delivered versus bounded in-process retries, request cancellation/restarts, and one logical event versus potentially repeated deliveries; obtain an approved delivery and consumer-idempotency guarantee before coding the publisher. If durability is required, specify transactional persistence and recovery without introducing an unapproved broker.
- [ ] T003 Update `specs/001-student-registration/plan.md`, `specs/001-student-registration/research.md`, and `specs/001-student-registration/quickstart.md` for the root module, `services/go.work` membership, root-level CI and shared contract paths, explicit `GOWORK`, env-supplied `DB_PATH`, stdlib testing, and the approved T001–T002 outcomes; remove the stale testify violation and nested-module instructions.
- [ ] T004 Initialize `go.mod` with module `lms-english/identity-service` and a compatible Go version of at least 1.23; pin only approved direct dependencies and create `go.sum` without adding testify.
- [ ] T005 Create `services/go.work` referencing the root module via `use ..`; verify `GOWORK` points to this absolute workspace path and `go list` recognizes the root module.
- [ ] T006 [P] Create `sqlc.yaml` with SQLite engine, `db/migrations` schema, `db/queries` input, and generated Go output under `internal/adapter/sqlite/sqlc/`.
- [ ] T007 [P] Create `.env.example` documenting `APP_ENV`, `HTTP_PORT=8081`, and `DB_PATH=./data/identity.db`; these are examples, not hardcoded runtime file paths, and must contain no secrets.
- [ ] T008 [P] Update `.gitignore` to exclude `data/*.db*`, including SQLite WAL and shared-memory sidecars.
- [ ] T009 After T001–T003, sync the reviewed `specs/001-student-registration/contracts/identity.openapi.yaml` to `repo:shared/contracts/identity.openapi.yaml`; verify the copies match before any handler is written.

**Checkpoint**: T001–T003 have accepted decisions; module, workspace, SQL configuration, and
contract are ready. No application code is implemented by this task-generation command.

## Phase 2: Database & Foundational Wiring

**Purpose**: Establish the SQLite schema and generated, parameterized query foundation.

- [ ] T010 Have a database owner author and review `db/migrations/00001_create_users_and_profiles.sql` with deterministic goose Up/Down, SQLite STRICT tables, role CHECK, unique normalized email, UTC timestamps, and profile foreign key with ON DELETE CASCADE. Apply [data-model.md](data-model.md) `User.ID`, `User.Email`, `User.PasswordHash`, `User.DisplayName`, `User.Role`, `User.CEFRLevel`, `User.CreatedAt`, `Profile.UserID`, `Profile.NativeLanguage`, `Profile.Goals`, `Profile.Timezone`, and `Profile.AvatarURL`.
- [ ] T011 Write parameterized `db/queries/users.sql` with `CreateUser :one` returning the inserted row, `GetUserByEmail :one`, and `CreateProfile :exec`; if T002 requires durable events, have a database owner author and review `db/migrations/00002_create_registration_events.sql` and add the approved persistence queries in `db/queries/registration_events.sql`.
- [ ] T012 Run `sqlc generate` from the module root using `sqlc.yaml`, inspect `internal/adapter/sqlite/sqlc/` for both tables and valid generated Go, and verify a second generation has no drift, including untracked generated files.
- [ ] T013 Implement `internal/adapter/sqlite/db.go` with `modernc.org/sqlite`, an environment-provided DB path, the requested WAL/synchronous/foreign-key/busy-timeout/cache PRAGMAs, startup verification, and `SetMaxOpenConns(1)`; propagate context and close resources.
- [ ] T014 Add `internal/adapter/sqlite/db_test.go` covering connection initialization and foreign keys; use a temporary file to verify WAL because an in-memory SQLite database cannot exercise file-backed WAL.

**Checkpoint**: Migration is reviewed, generated queries are reproducible, and the connection
foundation is testable. Do not rely on a pre-insert email lookup for uniqueness.

## Phase 3: User Story 1 — Register a New Student Account (P1, MVP)

**Goal**: A valid registration creates a student and empty profile atomically and signals the
registration without exposing a password.

**Independent test**: Register a unique valid address and compliant password; verify `201` and
`userId`, normalized email, `STUDENT`, an empty linked profile, and the T002-approved event
behavior. Also verify invalid and duplicate submissions cannot bypass core safeguards.

### Tests for User Story 1

- [ ] T015 [P] [US1] Write table-driven `internal/domain/registration_test.go` for user, role, profile, event, and baseline input invariants, including valid boundary values and the accepted T001 rules.
- [ ] T016 [P] [US1] Write `internal/usecase/register_user_test.go` with fakes for success, validation, hash failure, store rollback, context cancellation, event intent, and the T002 retry/failure behavior.
- [ ] T017 [P] [US1] Write `internal/adapter/sqlite/registration_store_test.go` using goose migrations and an isolated `:memory:` database with foreign keys enabled; assert atomic user/profile creation and rollback after profile failure.
- [ ] T018 [P] [US1] Write `internal/adapter/http/register_success_test.go` using `httptest` to assert the contract's `201` response, only `userId` in the body, malformed JSON `400`, and no password disclosure.
- [ ] T019 [P] [US1] Write `internal/adapter/http/visitor_limit_test.go` with a fake clock covering the approved T001 threshold, 15-minute window, 30-minute lockout, expiry, `429`, and no remaining-time disclosure.

### Inward Core

- [ ] T020 [US1] Implement stdlib-only entities and roles in `internal/domain/user.go`; follow [data-model.md](data-model.md) `User.ID`, `User.Email`, `User.PasswordHash`, `User.DisplayName`, `User.Role`, `User.CEFRLevel`, `User.CreatedAt`, plus its `Role` and `CEFRLevel` definitions.
- [ ] T021 [US1] Implement `internal/domain/profile.go` following [data-model.md](data-model.md) `Profile.UserID`, `Profile.NativeLanguage`, `Profile.Goals`, `Profile.Timezone`, and `Profile.AvatarURL`.
- [ ] T022 [US1] Implement `internal/domain/event.go` following [data-model.md](data-model.md) `UserRegisteredEvent.UserID`, `UserRegisteredEvent.Email`, `UserRegisteredEvent.Role`, and `UserRegisteredEvent.OccurredAt`.
- [ ] T023 [US1] Define registration and constitution sentinel errors in `internal/domain/errors.go`, including `ErrInvalidEmail`, `ErrWeakPassword`, `ErrEmailAlreadyExists`, `ErrInvalidCredentials`, `ErrForbidden`, and `ErrNotFound`.
- [ ] T024 [US1] Implement baseline email normalization/validation and password policy in `internal/domain/email.go` and `internal/domain/password.go`, following the approved T001 decisions and [data-model.md](data-model.md) `User.Email` and Validation Rules; import only the Go standard library.
- [ ] T025 [US1] Define `UserRegistrationStore`, `PasswordHasher`, `IDGenerator`, `Clock`, and `DomainEventPublisher` consumer ports in `internal/usecase/ports.go`, retaining the constitution's named output ports where applicable; use context and no adapter imports.
- [ ] T026 [US1] Implement `RegisterUserUseCase.Execute` in `internal/usecase/register_user.go`: validate, hash, generate ID, set defaults, create user/profile atomically, and apply T002 event semantics. Follow [data-model.md](data-model.md) `User.ID`, `User.Email`, `User.PasswordHash`, `User.DisplayName`, `User.Role`, `User.CEFRLevel`, `User.CreatedAt`, `Profile.UserID`, `Profile.NativeLanguage`, `Profile.Goals`, `Profile.Timezone`, `Profile.AvatarURL`, `UserRegisteredEvent.UserID`, `UserRegisteredEvent.Email`, `UserRegisteredEvent.Role`, and `UserRegisteredEvent.OccurredAt`.

### Adapters & HTTP Delivery

- [ ] T027 [US1] Implement `internal/adapter/security/bcrypt_hasher.go` and `internal/adapter/security/bcrypt_hasher_test.go` using the approved T001 password-length handling; hash and verify without logging or returning plaintext.
- [ ] T028 [US1] Implement `internal/adapter/sqlite/registration_store.go` with `BeginTx`, sqlc `WithTx`, user and profile inserts, rollback on failure, commit on success, and safe unique-error translation; include approved T002 event persistence in the same atomic boundary if required. Follow [data-model.md](data-model.md) `User.Email` and `Profile.UserID`.
- [ ] T029 [US1] Implement and test the T002-approved in-process event dispatcher and retry lifecycle in `internal/adapter/event/publisher.go` and `internal/adapter/event/publisher_test.go`, including shutdown and cancellation without losing accepted work under the approved guarantee.
- [ ] T030 [US1] Implement `internal/adapter/http/respond_error.go` using `errors.Is`/`errors.As`, the reviewed contract's JSON `code`, `message`, UTC `timestamp`, `path`, and `400`/`409`/`422`/`429`/`500` mappings; redact credentials and internal details.
- [ ] T031 [US1] Implement `internal/adapter/http/visitor_limit.go` using the approved T001 threshold and visitor identity policy; bound and synchronize state, do not trust unconfigured forwarding headers, and avoid exposing the remaining lockout time.
- [ ] T032 [US1] Implement `internal/adapter/http/register_handler.go` and `internal/adapter/http/router.go` with `chi.NewRouter()`, request ID, recovery, CORS, structured redacted logging, bounded JSON decoding, `POST /api/identity/auth/register`, and the reviewed contract's `201` response; use an injected use case and centralized errors.

### Entrypoint for the MVP

- [ ] T033 [US1] Implement `cmd/api/main.go` and `cmd/api/main_test.go`: load documented environment configuration, initialize and migrate SQLite, inject a stdlib `crypto/rand` UUID generator and other dependencies manually, configure HTTP timeouts, and gracefully stop the server and event worker.

**Checkpoint**: Run the US1 tests and demonstrate the complete safe registration path independently.

## Phase 4: User Story 2 — Reject Invalid Email (P2)

**Goal**: Explain the violated email rule without creating an account or emitting an event.

**Independent test**: Submit malformed email variants, including the resolved whitespace behavior;
each receives `422` with an actionable validation message and leaves no registration state.

- [ ] T034 [P] [US2] Add table-driven invalid-address and normalization cases to `internal/domain/email_invalid_test.go` using the accepted T001 rule.
- [ ] T035 [P] [US2] Add contract tests in `internal/adapter/http/register_invalid_email_test.go` for `422`, error envelope, and no account/event side effects.
- [ ] T036 [US2] Extend `internal/domain/email.go` to cover T034 while following [data-model.md](data-model.md) `User.Email` and Validation Rules as reconciled by T001; preserve sentinel matching.
- [ ] T037 [US2] Extend `internal/adapter/http/respond_error.go` and `internal/adapter/http/register_handler.go` to return the relevant email validation rule through the contract's `422` response; run T034–T035.

**Checkpoint**: US2 rejection is independently verifiable against an existing US1 foundation.

## Phase 5: User Story 3 — Reject Weak Password (P2)

**Goal**: Name the violated password rule and never persist or echo plaintext.

**Independent test**: Submit short, no-uppercase, and no-digit passwords; each receives `422`,
identifies the failing rule, and creates neither an account nor an event.

- [ ] T038 [P] [US3] Add table-driven cases in `internal/domain/password_invalid_test.go` for length, uppercase, digit, Unicode, and the approved T001 long-password policy.
- [ ] T039 [P] [US3] Add contract tests in `internal/adapter/http/register_weak_password_test.go` asserting `422`, rule details, no plaintext in the response, and no account/event side effects.
- [ ] T040 [US3] Extend `internal/domain/password.go` and `internal/adapter/security/bcrypt_hasher.go` to satisfy T038 and T001 without truncation, following [data-model.md](data-model.md) `User.PasswordHash` and Validation Rules.
- [ ] T041 [US3] Extend `internal/adapter/http/respond_error.go` and `internal/adapter/http/register_handler.go` for actionable weak-password errors; run T038–T039.

**Checkpoint**: US3 rejection is independently verifiable without weakening US1 or US2.

## Phase 6: User Story 4 — Reject an Already-Registered Email (P3)

**Goal**: Reject case-insensitive duplicates with a structured conflict and no account details.

**Independent test**: Register an address, then submit case variants and concurrent duplicates;
verify `409`, a stable error code, one account/profile, and no event for rejected attempts.

- [ ] T042 [P] [US4] Add SQLite integration tests in `internal/adapter/sqlite/registration_duplicate_test.go` for case-normalized duplicates, concurrent requests, email-specific constraint mapping, and rollback; distinguish an ID or profile constraint failure from an email conflict.
- [ ] T043 [P] [US4] Add HTTP contract tests in `internal/adapter/http/register_duplicate_test.go` for `409`, `email_already_exists`, and no sensitive account details.
- [ ] T044 [US4] Extend `internal/adapter/sqlite/registration_store.go` to translate only the confirmed users-email uniqueness violation to `domain.ErrEmailAlreadyExists` with cause preservation, enforcing [data-model.md](data-model.md) `User.Email`; run T042.
- [ ] T045 [US4] Extend `internal/adapter/http/respond_error.go` and `internal/adapter/http/register_handler.go` for the reviewed `409` contract; run T043.

**Checkpoint**: Duplicate handling is independently testable and concurrent-safe.

## Phase 7: Polish & Cross-Cutting Verification

**Purpose**: Complete the requested entrypoint/CI gate, operational documentation, and review.

- [ ] T046 Create `repo:.github/workflows/ci-identity.yml` with module-root working directory and explicit workspace, pinned sqlc and lint tooling, generated-file drift detection including untracked output, `golangci-lint run`, a CGO-disabled service build, and `go test -race ./...` in a race-capable environment.
- [ ] T047 Run `sqlc generate` from the module root and verify `internal/adapter/sqlite/sqlc/` has no generated drift; run `golangci-lint run`, `go test -race ./...`, and `CGO_ENABLED=0 go build ./cmd/api`, recording results in `specs/001-student-registration/quickstart.md` or the PR.
- [ ] T048 Execute the updated `specs/001-student-registration/quickstart.md` scenarios against a running service, using separate visitors or reset state so rate limiting does not mask the intended `201`, `400`, `409`, and `422` cases; separately verify `429` and recovery after expiry.
- [ ] T049 Review `internal/` and `cmd/api/main.go` for inward-only imports, redacted structured logs, context propagation, graceful shutdown, documented env configuration, exported Go documentation, and consistency with `repo:shared/contracts/identity.openapi.yaml`.

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1: T001–T003 are blocking design/governance gates; T004–T009 follow their applicable
  decisions. T006–T008 may run in parallel once the root module exists.
- Phase 2: T010 → T011 → T012; T013 → T014. The schema must be reviewed before generation.
- US1: Tests T015–T019 can be prepared in separate files after setup; domain → use case →
  adapters → entrypoint. T027, T028, T029, T030, and T031 use separate files but require the
  relevant completed ports and decisions; integrate them before T032–T033.
- US2 and US3: Each depends on US1's endpoint and core. Their test files can be prepared in
  parallel, but changes to shared handler/error files are sequential.
- US4: Depends on US1's store and endpoint; its integration and HTTP tests use separate files.
- Phase 7: Depends on all required story checkpoints; CI and local gates must agree.

### User Story Dependencies

```text
Setup + Foundation → US1 (safe MVP) → US2 (email cases)
                                ├→ US3 (password cases)
                                └→ US4 (duplicate cases)
US1 + US2 + US3 + US4 → CI / verification
```

The MVP includes baseline validation, uniqueness, lockout, and approved event handling; the
later stories deepen and verify their respective rejection paths. No story is released with
security or atomicity requirements omitted.

### Parallel Opportunities

- US1: T015–T019 write independent test files; after their interfaces exist, the hasher, store,
  event, error, and limiter adapters can be developed in separate files before integration.
- US2: T034 and T035 are separate domain and HTTP test files; integrate T036–T037 sequentially.
- US3: T038 and T039 are separate domain and HTTP test files; integrate T040–T041 sequentially.
- US4: T042 and T043 are independent SQLite and HTTP test files; integrate T044–T045 afterward.

## Implementation Strategy

### MVP First

1. Resolve T001–T003, then complete setup and the reviewed database foundation.
2. Implement US1's tested core, adapters, and entrypoint; validate the full registration path and
   required negative safeguards before demo or deployment.

### Incremental Delivery

Complete US2, US3, and US4 against the US1 foundation, exercising each independent test; then
complete Phase 7 and run the constitution's verification gate.

### Parallel Team Strategy

Separate owners can prepare independent tests and adapters once their dependencies exist. Avoid
concurrent edits to shared handler, domain, migration, and error files; merge and re-run all tests.

## Notes

- The reviewed OpenAPI document, not a fabricated endpoint implementation, is the contract-first
  source. Update it when T001–T002 alter its semantics before copying to `repo:shared/contracts/`.
- The user specified env examples for local use; the constitution prohibits hardcoded runtime file
  paths, so `DB_PATH` is supplied via environment rather than as a hardcoded program default.
- [bcrypt documentation](https://pkg.go.dev/golang.org/x/crypto/bcrypt#GenerateFromPassword)
  documents its 72-byte limit. The spec currently imposes no maximum, making T001 necessary.
- An IP-based limiter cannot guarantee that two visitors sharing an address are independent; the
  visitor-identity decision in T001 must address this limitation rather than claim otherwise.
