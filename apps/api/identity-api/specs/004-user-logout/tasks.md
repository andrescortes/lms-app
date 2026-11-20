---
description: "Implementation tasks for User Logout (CU-AUTH-04 / HU-19)"
---

# Tasks: User Logout (CU-AUTH-04 / HU-19)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: relative to `apps/api/identity-api/`. Tests use stdlib `testing` and `httptest`; no `testify`.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

**Wave 1 — independent (different files):**

- [x] **T001** [P] Merge the logout path and `bearerAuth` scheme into `contracts/identity.openapi.yaml`, then copy it to `repo:shared/contracts/identity.openapi.yaml` and confirm both parse and stay identical. · contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml

**Checkpoint**: Contract carries logout and the bearer scheme.

## Phase 2: Foundational

**Wave 1 — blocking sequence:**

- [x] **T002** Add `RevokeRefreshTokenByOwner` to `db/queries/tokens.sql`: revoke one row by `jti` only when it belongs to the given `user_id`, and return the row count so a miss is detectable. · db/queries/tokens.sql
- [x] **T003** Run `sqlc generate` and confirm regeneration is byte-stable. · internal/adapter/sqlite/sqlc/

**⟶ Wait for T003 to finish, then:**

- [x] **T004** Add `RevokeByJTI(ctx, userID, jti) error` to the `RefreshTokenRepository` port, documented to return `domain.ErrInvalidToken` for a missing, foreign, expired, or revoked token. · internal/usecase/ports.go

- [x] **T005** Implement `RevokeByJTI` in `internal/adapter/sqlite/refresh_token_store.go` over the scoped query, checking unrevoked and unexpired before revoking and reporting every refusal as `domain.ErrInvalidToken`. · internal/adapter/sqlite/refresh_token_store.go

**Checkpoint**: The scoped revoke exists and is wired to the port.

## Phase 3: User Story 1 — Sign Out and End the Session (P1) 🎯 MVP

**Goal**: A caller with a valid access token and their own refresh token receives `204` and the session ends.

**Independent Test**: Sign out and then attempt a refresh with the same token; the refresh is refused.

### Tests for User Story 1 (write first; confirm they fail)

- [x] **T006** [P] [US1] Add `internal/usecase/logout_test.go`: a valid owned token revokes and returns nil; a repository refusal surfaces as `domain.ErrInvalidToken`. · internal/usecase/logout_test.go

### Implementation for User Story 1

**Wave 1 — independent (different files):**

- [x] **T007** [US1] Implement `internal/usecase/logout.go`: `LogoutUseCase.Execute(ctx, LogoutCommand{UserID, RefreshToken})` rejects an empty token and delegates to `RevokeByJTI`. · internal/usecase/logout.go
- [x] **T008** [US1] Implement `internal/adapter/http/logout_handler.go`: decode the body, take the caller id from the request context, call the use case, and answer `204` with no body. · internal/adapter/http/logout_handler.go

**Checkpoint**: US1 works once the middlewares land.

## Phase 4: User Story 2 — Only the Session Owner Can End a Session (P1)

**Goal**: A foreign or unknown refresh token is refused and the other session survives.

- [x] **T009** [P] [US2] Add the cross-user tests: a foreign token is refused with `domain.ErrInvalidToken` at the use case and with `401 INVALID_TOKEN` at the handler, and the other user's row is untouched. · internal/usecase/logout_test.go, internal/adapter/http/logout_handler_test.go, internal/adapter/sqlite/logout_store_test.go

**Checkpoint**: US2 is independently verifiable.

## Phase 5: User Story 3 — Protected Routes Require a Valid Identity (P2)

**Goal**: Missing, malformed, expired, and wrong-method bearer tokens are refused with `401` before the handler runs.

- [x] **T010** [P] [US3] Add `internal/adapter/http/middleware_test.go`: bearer parsing, missing header, malformed token, expired token, wrong scheme, and that the identity reaches the handler and no handler runs on refusal. · internal/adapter/http/middleware_test.go
- [x] **T011** [US3] Implement `internal/adapter/http/middleware.go`: `Authenticate(verifier)` extracting `Authorization: Bearer <token>`, verifying with the pinned-HS256 provider, and putting `Claims{UserID, Role}` in the context. · internal/adapter/http/middleware.go
- [x] **T012** [US3] Register `POST /api/identity/auth/logout` inside an `Authenticate`-protected `chi` group and wire the use case in `cmd/api/main.go`. · internal/adapter/http/router.go, cmd/api/main.go

**Checkpoint**: US3 is independently verifiable.

## Phase 6: User Story 4 — Restricted Routes Refuse the Wrong Role (P3)

**Goal**: An authenticated caller whose role is not allowed receives `403 FORBIDDEN`.

- [x] **T013** [P] [US4] Add the `RequireRole` tests: allowed role passes, wrong role gets `403 FORBIDDEN`, empty allow list panics at construction. · internal/adapter/http/middleware_test.go
- [x] **T014** [US4] Implement `RequireRole(roles...)` and a `ClaimsFromContext` accessor that reports absence rather than returning a zero identity. · internal/adapter/http/middleware.go

**Checkpoint**: US4 is independently verifiable.

## Phase 7: Polish

- [x] **T015** Add `httptest` end-to-end cases for `204`, `400` (malformed body and empty token), and `401` (missing bearer, malformed bearer, foreign refresh token). · internal/adapter/http/logout_handler_test.go
- [x] **T016** Run the verification gate: `go test ./...`, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, and `sqlc generate` byte-stability.

## Dependencies & Execution Order

- **Phase 1** → **Phase 2** → story phases → **Phase 7**.
- **Phase 3** depends on Phase 2; **Phase 5** supplies the middleware US1 and US2 are exercised through.
- **Phase 6** is independent of the logout route and may run any time after Phase 2.

## Phase 8: Convergence

- [x] **T017** Remove the `internal/adapter/security` import from `internal/adapter/http/middleware.go` by declaring the verifier as a port the HTTP adapter consumes and having the security adapter satisfy it, so the HTTP adapter uses an injected interface rather than a sibling adapter per Constitution I (contradicts)
- [x] **T018** Drop the now-unneeded `JWTVerifier` bridge and the duplicate `Claims` type, keeping one identity type that the provider and the middleware share, per Constitution I / plan: Project Structure (contradicts)
