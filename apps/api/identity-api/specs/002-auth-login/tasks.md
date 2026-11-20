---
description: "Implementation tasks for User Login (CU-AUTH-02 / HU-02)"
---

# Tasks: User Login (CU-AUTH-02 / HU-02)

**Input**: Regenerated from the updated [spec.md](spec.md), [plan.md](plan.md),
[data-model.md](data-model.md), and [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: Paths are relative to the microservice root `apps/api/identity-api/` unless prefixed
`repo:` (repository root). The `go.mod` module `lms-english/identity-service` lives at the
microservice root; `services/go.work` references it via `use ..`, so commands set `GOWORK` to
that workspace's absolute path. Tests use Go stdlib `testing`/`httptest` with local fakes; do not
add `testify` (no ADR needed). Migration SQL is human-reviewed before `goose` applies it.

## Format: `[ID] [P?] [Story] Description`

- `[P]`: independent, different-file task; `[US1]`–`[US3]`: story from `spec.md`.

## Phase 1: Setup

**Purpose**: Contract-first gate and token secret before any handler work.

**Wave 1 — independent (different files):**

- [x] **T001** [P] Sync the login contract into the canonical copies: `specs/002-auth-login/contracts/identity.openapi.yaml` to `repo:shared/contracts/identity.openapi.yaml`, verify byte-identical before handler work. · specs/002-auth-login/contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml
- [x] **T002** [P] Add the `JWT_SECRET` example to `.env.example` and load it into the service config for token signing. · .env.example, cmd/api/main.go

**Checkpoint**: Canonical contract and token secret in place.

## Phase 2: Foundational

**Purpose**: Persistence and token infrastructure that blocks all stories.

**Wave 1 — foundational sequence (blocking):**

- [x] **T003** Write `db/migrations/00003_refresh_tokens.sql`: STRICT `refresh_tokens` table with unique `jti`, token hash, user id, expiry, and revocation state; goose Up/Down. · db/migrations/00003_refresh_tokens.sql
- [x] **T004** Write parameterized `db/queries/tokens.sql`: refresh token insert, load by jti, revoke. · db/queries/tokens.sql
- [x] **T005** Run `sqlc generate` and verify no drift in the generated sqlc package. · internal/adapter/sqlite/sqlc/
- [x] **T006** Implement `internal/adapter/sqlite/user_repository.go`: `UserRepository.FindByEmail` through the sqlc query, mapping a missing row to `domain.ErrNotFound`. · internal/adapter/sqlite/user_repository.go

**Checkpoint**: Reviewed migration, generated queries, and user lookup ready. No story work starts before this.

## Phase 3: User Story 1 — Sign In with Valid Credentials (P1) 🎯 MVP

**Goal**: Valid credentials return `200` with an access token (`15m`, sub `userId`, role claim) and a rotating refresh token (`7d`, persisted with unique jti).

**Independent Test**: Sign in with the correct email and password; receive `200` plus `accessToken` and `refreshToken`, never the password; the refresh token is persisted.

### Tests for User Story 1 (write first; confirm they fail)

- [x] **T007** [P] [US1] Write `internal/usecase/login_user_test.go` with fakes: valid credentials return tokens and persist the refresh token; trimmed/lowercased email accepted; wrong password and unknown email fail; malformed email fails; context cancellation propagates. · internal/usecase/login_user_test.go

### Implementation for User Story 1

**Wave 1 — independent (different files):**

- [x] **T008** [US1] Implement `internal/adapter/security/jwt_token_provider.go`: `TokenProvider` issuing an HS256 access token with `sub`, role, `iat`, and `exp` (`15m`), secret from configuration; compile-time interface proof. · internal/adapter/security/jwt_token_provider.go

**⟶ Wait for Wave 1 to finish, then:**

- [x] **T009** [US1] Implement `internal/usecase/login_user.go`: `LoginUserUseCase` over `UserRepository`, `PasswordVerifier`, `TokenProvider`, `RefreshTokenRepository`, and `Clock`; normalize and validate the email, look up the user, verify the password, issue the access token, store the refresh token, and return the tokens. · internal/usecase/login_user.go

**⟶ Wait for T009 to finish, then:**

- [x] **T010** [US1] Implement `internal/adapter/http/login_handler.go` and `router.go`: `POST /api/identity/auth/login`, `LoginRequest{email,password}`, `LoginResponse{accessToken,refreshToken}`, `200` on success; reuse the centralized error envelope. · internal/adapter/http/login_handler.go, internal/adapter/http/router.go

**⟶ Wait for T010 to finish, then:**

- [x] **T011** [US1] Wire `cmd/api/main.go`: login use case, token provider, refresh token store, and limiters; load `JWT_SECRET`; register the login route. · cmd/api/main.go

**Checkpoint**: US1 is independently functional and verified; MVP complete.

## Phase 4: User Story 2 — Sign In with Invalid Credentials (P2)

**Goal**: Invalid email or password returns `401` with the generic code and reveals nothing about account existence.

- [x] **T012** [P] [US2] Write the login failure tests: wrong password and unknown email fail with `domain.ErrInvalidCredentials`, no token or refresh row is written, and the response never discloses account existence; valid credentials still succeed after normalization (FR-001). · internal/usecase/login_user_test.go, internal/adapter/http/login_handler_test.go

**⟶ Wait for the US1 phase to finish, then:**

- [x] **T013** [US2] Map `domain.ErrInvalidCredentials` through the login use case to the centralized `401` outcome with the generic code; reuse the existing envelope code rather than adding a new one. · internal/usecase/login_user.go, internal/adapter/http/respond_error.go

**Checkpoint**: US2 is independently verifiable on the US1 foundation.

## Phase 5: User Story 3 — Rate Limit Sign In Attempts (P2)

**Goal**: At most 10 sign-in attempts per minute per IP; the 11th returns `429`.

- [x] **T014** [P] [US3] Write the visitor limiter tests for the login policy: attempts 1–10 allowed, 11th rejected with `429`, window reset, per-IP keying, other visitors unaffected, fail-open without a peer address. · internal/adapter/http/visitor_limit_test.go

**⟶ Wait for the US2 phase to finish, then:**

- [x] **T015** [US3] Implement the login attempt-limit policy in the visitor limiter and apply it to the login route. · internal/adapter/http/visitor_limit.go, internal/adapter/http/router.go

**Checkpoint**: US3 is independently verifiable.

## Phase 6: Polish, CI & Verification Gate

- [x] **T016** Run the full verification gate: `sqlc generate` with no drift, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, and `go test -race ./...`; confirm the OpenAPI contract copies are byte-identical; confirm the Success Criteria are covered.

**Checkpoint**: All user stories complete; verification gate satisfied.

## Dependencies & Execution Order

- **Phase 1** → **Phase 2**: contract and secret before foundation; foundation before stories.
- **Phase 2** → **Phase 3**: blocking; no story work starts before the foundational checkpoint.
- **Phase 3** → **Phase 4** → **Phase 5**: each story phase depends on the prior story phase.
- **Phase 6**: depends on all story checkpoints.

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2 (T001–T006).
2. Implement US1 (T007–T011): token provider, login use case, handler, wiring.
3. **STOP and VALIDATE** US1 independently, then proceed.

### Incremental Delivery

Add US2 and US3 in priority order, each with its independent tests, then finish Phase 6 and the verification gate.

### Parallel Team Strategy

Independent test tasks and adapter tasks can be prepared once the foundational artifacts exist; avoid concurrent edits to shared domain, handler, migration, and error files.

## Notes

- The reviewed OpenAPI contract is the source of truth; the feature and shared copies must match (T001).
- `DB_PATH` is an environment example, not a hardcoded default (constitution IV).
- The login rate limit reuses the per-visitor limiter machinery rather than a new type (research Decision 3).
- Write each story's tests first and confirm they fail before implementation; commit after each task or logical group.
