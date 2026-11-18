---
description: "Implementation tasks for Student Registration (CU-AUTH-01 / HU-01)"
---

# Tasks: Student Registration (CU-AUTH-01 / HU-01)

**Input**: Regenerated from the updated [spec.md](spec.md), [plan.md](plan.md),
[data-model.md](data-model.md), and [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Clarifications incorporated (2026-11-18)**: one logical `UserRegistered` event recorded in a
SQLite outbox with restart-safe, retrying, duplicate-tolerant delivery; sixth attempt in a
15-minute window triggers the 30-minute lockout; leading/trailing email whitespace is trimmed;
passwords are 8–72 characters inclusive; only validation and duplicate-conflict rejections name
the specific rule. All five have matching implementation and test tasks below.

**Paths**: Paths are relative to the microservice root `apps/api/identity-api/` unless prefixed
`repo:` (repository root). The `go.mod` module `lms-english/identity-service` lives at the
microservice root; `services/go.work` references it via `use ..`, so commands set `GOWORK` to
that workspace's absolute path. Tests use Go stdlib `testing`/`httptest` with local fakes; do not
add `testify` (no ADR needed). Migration SQL is human-reviewed before `goose` applies it.

## Format: `[ID] [P?] [Story] Description`

- `[P]`: independent, different-file task; `[US1]`–`[US4]`: story from `spec.md`.
- Setup, foundational, and polish phases carry no story label.

## Phase 1: Setup & Greenfield Scaffolding

**Purpose**: Resolve the remaining layout drift, initialize the root module, and publish the contract.

- [ ] T001 Update `specs/001-student-registration/quickstart.md`: replace the nested-service bootstrap commands with `go.mod` at the microservice root and `services/go.work` using `use ..`; remove the stale testify-ADR note; update scenarios for the 8–72 password and sixth-attempt lockout. `plan.md` is already aligned.
- [ ] T002 Initialize `go.mod` at the microservice root: module `lms-english/identity-service`, Go 1.23+ with `toolchain`, `CGO_ENABLED=0` build; pin only approved direct dependencies (`chi/v5`, `chi/cors`, `modernc.org/sqlite`, `golang.org/x/crypto`, `golang-jwt/jwt/v5`, `pressly/goose/v3`) and create `go.sum`; do not add testify.
- [ ] T003 [P] Create `services/go.work` referencing the root module via `use ..`; verify `GOWORK` points to its absolute path and `go list ./...` resolves the module.
- [ ] T004 [P] Create `sqlc.yaml`: `engine: "sqlite"`, `schema: "db/migrations"`, `queries: "db/queries"`, generated Go under `internal/adapter/sqlite/sqlc/`.
- [ ] T005 [P] Create `.env.example` documenting `APP_ENV`, `HTTP_PORT=8081`, `DB_PATH=./data/identity.db`; no real secrets or absolute paths.
- [ ] T006 [P] Add `data/*.db*` to `.gitignore` (including `-wal`/`-shm` sidecars).
- [ ] T007 Sync `specs/001-student-registration/contracts/identity.openapi.yaml` to `repo:shared/contracts/identity.openapi.yaml`; verify byte-identical before any handler work (constitution contract-first gate).
- [ ] T008 [P] Scaffold directories: `cmd/api/`, `internal/domain/`, `internal/usecase/`, `internal/adapter/{http,sqlite,security,event}/`, `db/{migrations,queries}/`.
- [ ] T009 [P] Add `.golangci.yml` at the microservice root enabling `bodyclose`, `sqlclosecheck`, `errcheck`, `govet`, `staticcheck` per the security skill.
- [ ] T010 [P] Add a local copy of the contract at `contracts/identity.openapi.yaml` for handler authors, matching the feature and shared copies.

**Checkpoint**: Root module, workspace, tooling, and canonical contract ready.

## Phase 2: Database & Foundational Wiring

**Purpose**: Reviewed migrations, generated queries, and SQLite connection foundation (blocking).

- [ ] T011 Have a database owner author and review `db/migrations/00001_create_users_and_profiles.sql`: goose Up/Down; `STRICT` `users` and `profiles`; role `CHECK`; `UNIQUE` normalized email; `ON DELETE CASCADE`; ISO-8601 UTC `TEXT` timestamps; Down drops `profiles` before `users`. Apply [data-model.md](data-model.md) `User.ID`, `User.Email`, `User.PasswordHash`, `User.DisplayName`, `User.Role`, `User.CEFRLevel`, `User.CreatedAt`, `Profile.UserID`, `Profile.NativeLanguage`, `Profile.Goals`, `Profile.Timezone`, `Profile.AvatarURL`.
- [ ] T012 Have a database owner author and review `db/migrations/00002_create_registration_events.sql`: `STRICT` `registration_events` outbox with `id`, `event_type`, `payload`, `status` (`pending|delivered`), `attempts`, `created_at`, `delivered_at`; independent Down.
- [ ] T013 Write parameterized `db/queries/users.sql`: `CreateUser :one`, `GetUserByEmail :one`, `CreateProfile :exec`.
- [ ] T014 Write parameterized `db/queries/events.sql`: `CreateEvent :exec`, `ListPendingEvents :many`, `MarkEventDelivered :exec`; all context-aware, no string concatenation.
- [ ] T015 Run `sqlc generate`; inspect `internal/adapter/sqlite/sqlc/` for `users`, `profiles`, and `registration_events` models; verify a second run produces no drift, including untracked files.
- [ ] T016 Implement `internal/adapter/sqlite/db.go`: `modernc.org/sqlite`, env-supplied `DB_PATH`, DSN `_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=cache_size(-64000)`, `SetMaxOpenConns(1)`, startup pragma verification.
- [ ] T017 Add `internal/adapter/sqlite/db_test.go`: connection init, foreign-key enforcement, and file-backed WAL verification on a temporary database.

**Checkpoint**: Reviewed schema, reproducible generated queries, testable connection layer. No story work starts before this.

## Phase 3: User Story 1 — Register a New Student Account (P1) 🎯 MVP

**Goal**: A valid registration creates a student, an empty profile, and one durable event atomically, with inputs normalized and passwords protected.

**Independent Test**: Register a unique email (with surrounding spaces) and a compliant password; verify `201` + `userId`, normalized stored email, `STUDENT`, empty profile, exactly one outbox event; verify attempts 1–5 are accepted and the 6th returns `429`.

### Tests for User Story 1 (write first; confirm they fail)

- [ ] T018 [P] [US1] Write table-driven `internal/domain/registration_test.go`: entities/roles and validation boundaries — trim+lowercase email, 8/72 accepted and 7/73 rejected, uppercase and digit rules.
- [ ] T019 [P] [US1] Write `internal/usecase/register_user_test.go` with fakes: success, invalid email, weak/too-long password, hash failure, store rollback, unique conflict, event recorded, publisher failure not failing registration, context cancellation; assert Execute returns without invoking the publisher/hook (delivery is the worker's responsibility).
- [ ] T020 [P] [US1] Write `internal/adapter/sqlite/registration_store_test.go` on goose-migrated `:memory:` DB: atomic user+profile+outbox insert, rollback on profile failure, `UNIQUE` → `ErrEmailAlreadyExists`, exactly one outbox row.
- [ ] T021 [P] [US1] Write `internal/adapter/event/publisher_test.go`: claim pending, dispatch to hook, mark delivered, retry on failure, reprocess pending after simulated restart, and no duplicate downstream effect for a repeated event id; assert a slow/failing hook does not delay or fail registration.
- [ ] T022 [P] [US1] Write `internal/adapter/http/register_success_test.go` with `httptest`: `201`, only `userId` body, `Location` header, no password disclosure.
- [ ] T023 [P] [US1] Write `internal/adapter/http/visitor_limit_test.go` with a fake clock: attempts 1–5 allowed, 6th → `429`, no remaining-time disclosure, no `Retry-After`, other visitor unaffected, 30-minute expiry, window boundary.

### Implementation for User Story 1

- [ ] T024 [P] [US1] Implement stdlib-only entities in `internal/domain/user.go`: `Role` enum (`RoleUnknown=0`, `RoleStudent=1`, ...), `CEFRLevel`, `User`. Cite [data-model.md](data-model.md) → `User.ID`, `User.Email`, `User.PasswordHash`, `User.DisplayName`, `User.Role`, `User.CEFRLevel`, `User.CreatedAt` and its `Role`/`CEFRLevel` definitions.
- [ ] T025 [P] [US1] Implement `internal/domain/profile.go`: `Profile` struct. Cite [data-model.md](data-model.md) → `Profile.UserID`, `Profile.NativeLanguage`, `Profile.Goals`, `Profile.Timezone`, `Profile.AvatarURL`.
- [ ] T026 [P] [US1] Implement `internal/domain/event.go`: `UserRegisteredEvent`. Cite [data-model.md](data-model.md) → `UserRegisteredEvent.UserID`, `UserRegisteredEvent.Email`, `UserRegisteredEvent.Role`, `UserRegisteredEvent.OccurredAt`.
- [ ] T027 [P] [US1] Define sentinel errors in `internal/domain/errors.go`: `ErrInvalidEmail`, `ErrWeakPassword`, `ErrEmailAlreadyExists`, plus `ErrInvalidCredentials`, `ErrForbidden`, `ErrNotFound`.
- [ ] T028 [P] [US1] Implement `internal/domain/email.go`: trim + lowercase `NormalizeEmail`, package-level compiled `regexp` format check in `ValidateEmail`, `ErrInvalidEmail`.
- [ ] T029 [P] [US1] Implement `internal/domain/password.go`: `ValidatePasswordPolicy` — 8–72 characters counted as runes, ≥1 uppercase, ≥1 digit, `ErrWeakPassword`; no plaintext retained (cite [data-model.md](data-model.md) → Password Validation Rules).
- [ ] T030 [US1] Define consumer ports in `internal/usecase/ports.go`: `UserRegistrationStore`, `PasswordHasher`, `IDGenerator`, `Clock`, `DomainEventPublisher`, plus the constitution-mandated `UserRepository`, `PasswordVerifier`, `TokenProvider`, `RefreshTokenRepository`; domain + stdlib imports only.
- [ ] T031 [US1] Implement `RegisterUserUseCase.Execute(ctx, cmd)` in `internal/usecase/register_user.go`: normalize/validate, hash, generate ID, default `displayName`, and call the store port once to atomically write user, profile, and the outbox event row. The use case MUST NOT invoke downstream hooks or wait for delivery — the outbox worker (T034/T038) dispatches after commit, so a slow or failing hook cannot delay or fail registration. Propagate `context.Context` and wrap with `fmt.Errorf("...: %w", err)`.
- [ ] T032 [P] [US1] Implement `internal/adapter/security/bcrypt_hasher.go`: `BcryptHasher` hashing/verifying 8–72-character passwords with no silent truncation.
- [ ] T033 [US1] Implement `internal/adapter/sqlite/registration_store.go`: `BeginTx` + `Queries.WithTx`, `CreateUser` → `CreateProfile` → `CreateEvent`, commit/rollback, translate the users-email `UNIQUE` violation to `domain.ErrEmailAlreadyExists`; add a compile-time interface assertion. Cite [data-model.md](data-model.md) → `User.Email`, `Profile.UserID`.
- [ ] T034 [US1] Implement `internal/adapter/event/publisher.go`: outbox-backed `DomainEventPublisher` with in-process hooks, backoff retry checking `ctx.Err()`, startup recovery of `pending` rows, `MarkEventDelivered`, and stable event-id idempotency at consumers (cite [data-model.md](data-model.md) → registration_events columns and `UserRegisteredEvent`).
- [ ] T035 [P] [US1] Implement `internal/adapter/http/respond_error.go`: centralized `respondError(w, r, err)` with `{"code","message","timestamp","path"}`, `errors.Is`/`errors.As` mapping (400/409/422/429/500), redacted messages; validation/conflict name the rule, other outcomes use standard codes (SC-004 scope).
- [ ] T036 [P] [US1] Implement `internal/adapter/http/visitor_limit.go`: bounded, mutex-protected in-memory map keyed on the peer host from `r.RemoteAddr`; first five attempts in 15 minutes allowed, sixth → 30-minute lockout; no remaining time, no `Retry-After`; untrusted forwarding headers ignored; expired/pruned entries bounded.
- [ ] T037 [US1] Implement `internal/adapter/http/register_handler.go` and `router.go`: `chi.NewRouter()` with `middleware.RequestID`, `middleware.Recoverer`, `log/slog`, `chi/cors`; `POST /api/identity/auth/register` `http.HandlerFunc`; bounded `MaxBytesReader` + one `DisallowUnknownFields` decode; `201` with only `userId` and `Location`; inject the use case and `respondError`.
- [ ] T038 [US1] Implement `cmd/api/main.go` and `cmd/api/main_test.go`: load documented env config, initialize/migrate SQLite, manual constructor DI, start the outbox retry worker, explicit `http.Server` timeouts, graceful shutdown of server and worker.

**Checkpoint**: US1 is independently functional and verified; MVP complete.

## Phase 4: User Story 2 — Reject Invalid Email (P2)

**Goal**: Malformed addresses get a `422` naming the email rule, with no account or event.

**Independent Test**: Submit addresses with no `@`, no dot, trailing dot, or empty local part; each returns `422 validation_error` naming the email rule and creates no account/outbox row.

- [ ] T039 [P] [US2] Add table-driven cases in `internal/domain/email_invalid_test.go` for missing `@`/dot, trailing dot, empty local part, whitespace-trimmed-valid, and disallowed characters.
- [ ] T040 [P] [US2] Add `internal/adapter/http/register_invalid_email_test.go`: `422`, `validation_error`, email rule named, no account/outbox side effects.
- [ ] T041 [US2] Extend `internal/domain/email.go` to satisfy T039 while honoring [data-model.md](data-model.md) → `User.Email` and its Email Validation Rules; preserve `errors.Is` sentinel matching.
- [ ] T042 [US2] Extend `internal/adapter/http/respond_error.go` and `register_handler.go` to return the email validation rule in the `422` message; run T039–T040.

**Checkpoint**: US2 independently verifiable on the US1 foundation.

## Phase 5: User Story 3 — Reject Weak Password (P2)

**Goal**: A password failing any rule gets a `422` naming that rule; plaintext never appears anywhere.

**Independent Test**: Submit passwords of length 7 and 73, no uppercase, and no digit; each returns `422` naming the rule, with no account/outbox row and no plaintext in any response.

- [ ] T043 [P] [US3] Add table-driven cases in `internal/domain/password_invalid_test.go` for <8, >72, no uppercase, no digit, exactly 72 valid, and symbols/spaces/accented characters allowed.
- [ ] T044 [P] [US3] Add `internal/adapter/http/register_weak_password_test.go`: `422` naming length/uppercase/digit, no plaintext echoed, no account/outbox side effects.
- [ ] T045 [US3] Extend `internal/domain/password.go` and `bcrypt_hasher.go` for T043 boundaries per [data-model.md](data-model.md) → Password Validation Rules; never truncate.
- [ ] T046 [US3] Extend `respond_error.go` and `register_handler.go` for actionable weak-password messages; run T043–T044.

**Checkpoint**: US3 independently verifiable without weakening US1/US2.

## Phase 6: User Story 4 — Reject an Already-Registered Email (P3)

**Goal**: Case-insensitive duplicates get `409 email_already_exists` with no account details and no event for the loser.

**Independent Test**: Register an address, then submit case variants and concurrent duplicates; verify `409`, a stable code, one account/profile/outbox row total, and no event for rejected attempts.

- [ ] T047 [P] [US4] Add `internal/adapter/sqlite/registration_duplicate_test.go`: case-normalized duplicate, concurrent duplicate (one success/one conflict), email-specific constraint mapping, exactly one outbox row (winner only), rollback on failure.
- [ ] T048 [P] [US4] Add `internal/adapter/http/register_duplicate_test.go`: `409`, `email_already_exists`, no existing-account details, no password.
- [ ] T049 [US4] Extend `registration_store.go` so only the confirmed users-email uniqueness violation maps to `domain.ErrEmailAlreadyExists` with cause preserved; cite [data-model.md](data-model.md) → `User.Email`; run T047.
- [ ] T050 [US4] Extend `respond_error.go` and `register_handler.go` for the reviewed `409` contract; run T048.

**Checkpoint**: Duplicate handling is independently testable and concurrency-safe.

## Phase 7: Polish, CI & Verification Gate

**Purpose**: Encode the constitution's gate and validate clarified end-to-end behavior.

- [ ] T051 Create `repo:.github/workflows/ci-identity.yml`: module-root working dir with explicit `GOWORK`, pinned `sqlc`/lint tooling, `sqlc generate` drift check (including untracked), `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, and `go test -race ./...` in a race-capable environment.
- [ ] T052 [P] Add a timed acceptance check for SC-001 ("under 3 seconds under normal conditions") in `internal/adapter/http/register_success_test.go` with documented normal conditions.
- [ ] T053 Add an SC-006 audit check in `internal/adapter/sqlite/registration_store_test.go` verifying no plaintext password in stored data, event payloads, error bodies, or logs across success/validation/conflict outcomes.
- [ ] T054 Run the full verification gate locally — `sqlc generate` (no drift), `golangci-lint run`, `go test -race ./...`, `CGO_ENABLED=0 go build ./cmd/api` — and document results in the pull request (constitution governance requires PR-recorded verification).
- [ ] T055 Execute the updated `quickstart.md` scenarios (201/409/422/400/429) against a running service, using fresh visitors or reset state so the 429 lockout does not mask other cases; confirm the 8–72 password and sixth-attempt boundaries against the contract.
- [ ] T056 [P] Review `internal/` and `cmd/` for inward-only imports, redacted structured logs, `context.Context` propagation, graceful shutdown, env-only config, and exported-symbol godoc comments.
- [ ] T057 [P] Verify the feature, module-local, and `repo:shared/contracts/` OpenAPI copies remain byte-identical after all changes.
- [ ] T058 Record the deferred service-wide mandates (JWT/RBAC middleware, JWT signing/validation, token issuance) in the project backlog with a follow-up feature reference; do not implement them in this feature (Principle I explicit deferral).

**Checkpoint**: All user stories complete; constitution verification gate satisfied.

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001–T007 sequential where noted; T003–T006, T008–T010 parallel after T002.
- **Foundational (Phase 2)**: reviewed migrations (T011–T012) → queries (T013–T014) → `sqlc generate` (T015) → DB init (T016–T017). BLOCKS all stories.
- **US1 (Phase 3)**: tests T018–T023 first; domain T024–T029; ports/usecase T030–T031; adapters T032–T037; entrypoint T038.
- **US2/US3/US4 (Phases 4–6)**: each depends on US1; their test files are independent, shared-file edits are sequential.
- **Polish (Phase 7)**: depends on all story checkpoints.

### User Story Dependencies

```text
Setup + Foundation → US1 (safe MVP) → US2 (email)
                                 ├→ US3 (password)
                                 └→ US4 (duplicate)
US1 + US2 + US3 + US4 → CI / verification gate
```

### Parallel Opportunities

- US1: T018–T023 are separate test files; T024–T026 domain entities and T032/T035/T036 adapters are independent files.
- US2: T039 and T040 (separate test files). US3: T043 and T044. US4: T047 and T048.
- Setup and polish tasks marked `[P]` are different-file and independent.

## Implementation Strategy

### MVP First

1. Complete Setup and Foundational (T001–T017).
2. Implement US1 (T018–T038): normalized inputs, atomic user/profile/outbox write, bcrypt, durable retrying event, centered errors, lockout, entrypoint.
3. **STOP and VALIDATE** US1 independently, then proceed.

### Incremental Delivery

Add US2, US3, US4 in order, each with its independent tests, then finish Phase 7 and the verification gate.

### Parallel Team Strategy

Separate owners can prepare independent test files and adapters once foundational artifacts exist; avoid concurrent edits to shared domain, handler, migration, and error files.

## Notes

- The reviewed OpenAPI contract is the source of truth; shared and module-local copies must match (T007/T010/T057).
- `DB_PATH` is an environment example, not a hardcoded default (constitution IV).
- The outbox (T012/T034) needs no new dependency and no ADR; testify is excluded entirely.
- No JWT/RBAC or token-issuance tasks are included: registration is public and those are separate service-wide capabilities; keep this feature's scope bounded.
- Write each story's tests first and confirm they fail before implementation; commit after each task or logical group.
