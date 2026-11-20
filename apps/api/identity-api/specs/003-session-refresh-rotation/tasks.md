---
description: "Implementation tasks for Session Refresh Rotation (CU-AUTH-03 / HU-03)"
---

# Tasks: Session Refresh Rotation (CU-AUTH-03 / HU-03)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: relative to `apps/api/identity-api/` unless prefixed `repo:`. Tests use stdlib `testing`, `httptest`, and local fakes; do not add `testify`.

## Format: `[ID] [P?] [Story] Description`

- `[P]`: independent, different-file task; `[US1]`–`[US4]`: story from `spec.md`.

## Phase 1: Setup

**Wave 1 — independent (different files):**

- [x] **T001** [P] Sync the refresh contract into the canonical copy: `specs/003-session-refresh-rotation/contracts/identity.openapi.yaml` into `repo:shared/contracts/identity.openapi.yaml`, verifying the result is valid YAML. · specs/003-session-refresh-rotation/contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml

**Checkpoint**: Canonical contract in place.

## Phase 2: Foundational

**Wave 1 — blocking sequence:**

- [x] **T002** Add the rotation queries to `db/queries/tokens.sql`: revoke every active refresh token for a user, and reuse the existing insert and lookup; keep compatibility with the existing `refresh_tokens` schema. · db/queries/tokens.sql
- [x] **T003** Run `sqlc generate` and verify no drift in the generated package. · internal/adapter/sqlite/sqlc/

**⟶ Wait for T003 to finish, then:**

- [x] **T004** Add `Rotate` to the `RefreshTokenRepository` port, reporting whether the presented token was rotated, spent, expired, or unknown. · internal/usecase/ports.go

**Checkpoint**: Queries, generated code, and the port exist.

## Phase 3: User Story 1 — Refresh a Session with a Valid Token (P1) 🎯 MVP

**Goal**: A valid token is revoked and replaced in one step, and a new access token is issued.

**Independent Test**: Refresh once and receive both tokens; presenting the spent token again is refused.

### Tests for User Story 1 (write first; confirm they fail)

- [x] **T005** [P] [US1] Add `internal/usecase/refresh_token_test.go`: a valid token returns a new pair and the store reports a rotation; expired, unknown, and spent tokens each fail. · internal/usecase/refresh_token_test.go

### Implementation for User Story 1

**Wave 1 — independent (different files):**

- [x] **T006** [US1] Implement `Rotate` in `internal/adapter/sqlite/refresh_token_store.go`: one `*sql.Tx` that reads by hash, checks expiry and revocation, revokes the presented row, and inserts the replacement. · internal/adapter/sqlite/refresh_token_store.go
- [x] **T007** [US1] Implement `internal/usecase/refresh_token.go`: `RefreshTokenUseCase.Execute` looks up the user, rotates, and issues a new 15m access token plus a new 7d refresh token. · internal/usecase/refresh_token.go

**Checkpoint**: US1 is independently functional and verified; MVP complete.

## Phase 4: User Story 2 — Refuse Invalid, Expired, and Revoked Tokens (P2)

**Goal**: Every bad token returns `401 INVALID_TOKEN` and no tokens.

- [x] **T008** [P] [US2] Add store tests for the exact-format expiry comparison and the unknown-hash path. · internal/adapter/sqlite/refresh_store_test.go
- [x] **T009** [US2] Map the invalid-token outcome into the centralized error envelope with the code `INVALID_TOKEN`. · internal/adapter/http/respond_error.go

**Checkpoint**: US2 is independently verifiable.

## Phase 5: User Story 3 — Revoke the Whole Chain on Replay (P2)

**Goal**: A spent token revokes every active token for that user.

- [x] **T010** [P] [US3] Add the chain-revocation test: two active tokens, spend one, replay it, and assert both are revoked afterwards. · internal/adapter/sqlite/refresh_store_test.go
- [x] **T011** [US3] Implement the replay branch in `Rotate`: when the matched row is revoked, revoke every active row for the user inside the same transaction and report the reuse. · internal/adapter/sqlite/refresh_token_store.go

**Checkpoint**: US3 is independently verifiable.

## Phase 6: User Story 4 — HTTP Surface and Rate Limit (P2)

**Goal**: `POST /api/identity/auth/refresh` answers `200`, `400`, `401`, and `429`.

- [x] **T012** [P] [US4] Add `httptest` handler tests for `200`, `400`, `401`, and `429` through the router. · internal/adapter/http/refresh_handler_test.go
- [x] **T013** [US4] Implement `internal/adapter/http/refresh_handler.go` and register the route with its own `httprate` bucket. · internal/adapter/http/refresh_handler.go, internal/adapter/http/router.go

**⟶ Wait for T013 to finish, then:**

- [x] **T014** [US4] Wire `RefreshTokenUseCase` in `cmd/api/main.go`. · cmd/api/main.go

**Checkpoint**: US4 is independently verifiable.

## Phase 7: Polish

- [x] **T015** Add `Verify` to `internal/adapter/security/jwt_token_provider.go` with `jwt.WithValidMethods([]string{"HS256"})`, returning `domain.ErrInvalidToken`, and its test. · internal/adapter/security/jwt_token_provider.go, internal/adapter/security/jwt_token_provider_test.go
- [x] **T016** Run the verification gate: `sqlc generate` with no drift, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, `go test ./...`, and confirm the contract copies match.

## Dependencies & Execution Order

- **Phase 1** → **Phase 2** → **Phase 3** → **Phase 4** → **Phase 5** → **Phase 6** → **Phase 7**, each blocking the next.
- Within Phase 3, T006 and T007 touch different files and can proceed in either order.

## Phase 8: Convergence

- [x] **T017** Pass the replacement token's jti from `RefreshTokenUseCase` into `Rotate` instead of generating it inside the SQLite adapter, so identifier policy stays in the application layer per Constitution V (unrequested)
- [x] **T018** Construct `adaptersqlite.NewUserRepository(db)` once in `cmd/api/main.go` and share it between the login and refresh use cases per plan: Project Structure (unrequested)

