---
description: "Implementation tasks for User Profile Management (CU-AUTH-05 / HU-18)"
---

# Tasks: User Profile Management (CU-AUTH-05 / HU-18)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: relative to `apps/api/identity-api/`. Tests use stdlib `testing` and `httptest`; no `testify`.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

**Wave 1 — independent (different files):**

- [x] **T001** [P] Merge the two `/api/identity/users/me` paths and their schemas into `contracts/identity.openapi.yaml`, copy to `repo:shared/contracts/identity.openapi.yaml`, and confirm both parse with equal paths and schemas. · contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml

**Checkpoint**: Canonical contract carries the profile routes.

## Phase 2: Foundational

**Wave 1 — blocking sequence:**

- [x] **T002** Add to `db/queries/users.sql`: `GetUserWithProfileByID` (joined read), `UpdateUserDisplayName`, `UpdateUserPasswordHash`, and `UpdateProfileFields`. · db/queries/users.sql
- [x] **T003** Run `sqlc generate` and confirm byte-stability. · internal/adapter/sqlite/sqlc/
- [x] **T004** Add a `ProfileStore` port with `GetUserProfile(ctx, userID)` and `UpdateUserProfile(ctx, change)` where the change carries optional account and profile values. · internal/usecase/ports.go

**⟶ Wait for T004 to finish, then:**

- [x] **T005** Implement `internal/adapter/sqlite/profile_store.go`: the joined read mapping `sql.ErrNoRows` to `domain.ErrNotFound`, and a transactional update writing only the fields the change names. · internal/adapter/sqlite/profile_store.go

**Checkpoint**: Storage exists behind a port.

## Phase 3: User Story 1 — View My Account and Profile (P1) 🎯 MVP

**Goal**: A signed-in learner reads their account and profile with no password material.

**Independent Test**: Read the profile and assert every account and profile field, and that no password value appears anywhere in the response.

### Tests for User Story 1 (write first; confirm they fail)

- [x] **T006** [P] [US1] Add `internal/usecase/get_profile_test.go` and `internal/adapter/sqlite/profile_store_test.go`: a registered user reads back with empty profile fields, an unknown id reports `domain.ErrNotFound`, and no hash is reachable through the read model. · internal/usecase/get_profile_test.go, internal/adapter/sqlite/profile_store_test.go

### Implementation for User Story 1

- [x] **T007** [US1] Implement `internal/usecase/get_profile.go`: `GetProfileUseCase.Execute(ctx, userID)` returning the account and profile. · internal/usecase/get_profile.go
- [x] **T008** [US1] Implement the read handler in `internal/adapter/http/profile_handler.go` and register `GET /api/identity/users/me` in the authenticated group. · internal/adapter/http/profile_handler.go, internal/adapter/http/router.go

**Checkpoint**: US1 is independently functional; MVP complete.

## Phase 4: User Story 2 — Change a Profile Field (P1)

**Goal**: Present fields change, absent fields stay, `null` clears.

**Independent Test**: Update only goals, then read back; every other field is unchanged. Send `null` for a field and read back; it is empty.

### Tests for User Story 2 (write first; confirm they fail)

- [x] **T009** [P] [US2] Add partial-update tests across the use case and store: one field changed with the rest intact, an explicit null clearing a field, and an empty change leaving every field as it was. · internal/usecase/update_profile_test.go, internal/adapter/sqlite/profile_store_test.go

### Implementation for User Story 2

- [x] **T010** [US2] Implement `internal/usecase/update_profile.go` for the profile and display-name part of `UpdateProfileCommand`: apply only the named fields, refuse a blank display name. · internal/usecase/update_profile.go

**⟶ Wait for T010 to finish, then:**

- [x] **T011** [US2] Implement the `PATCH` handler in `internal/adapter/http/profile_handler.go` with pointer-based decoding so absent and null differ, and register the route. · internal/adapter/http/profile_handler.go, internal/adapter/http/router.go

**Checkpoint**: US2 is independently verifiable.

## Phase 5: User Story 3 — Change My Password Safely (P1)

**Goal**: A password change needs the current password, meets the policy, and stores only the hash.

**Independent Test**: Change the password with the correct current one, then sign in with both; only the new one works. Missing current, wrong current, and weak new each refuse and leave the old password working.

### Tests for User Story 3 (write first; confirm they fail)

- [x] **T012** [P] [US3] Add password-change tests: valid change, missing current, wrong current, weak new password, and that a refusal leaves the stored hash byte-identical. · internal/usecase/update_profile_test.go, internal/adapter/sqlite/profile_store_test.go

### Implementation for User Story 3

- [x] **T013** [US3] Add the password branch to `internal/usecase/update_profile.go`: require and verify the current password before validating and hashing the new one. · internal/usecase/update_profile.go

**Checkpoint**: US3 is independently verifiable.

## Phase 6: User Story 4 — Only I Can See and Change My Profile (P1)

**Goal**: Both routes refuse a missing or invalid bearer credential.

- [x] **T014** [P] [US4] Add `httptest` cases through the router: no bearer and an expired bearer on both routes return `401`, and the handler is never reached. · internal/adapter/http/profile_handler_test.go

**Checkpoint**: US4 is independently verifiable.

## Phase 7: Polish

- [x] **T015** Add the `WEAK_PASSWORD` code to `respond_error.go` for the weak-password refusal, keeping the existing validation detail. · internal/adapter/http/respond_error.go
- [x] **T016** Add `httptest` cases for `200`, `400`, `401`, `404`, and `422` on both routes, and confirm no response body contains a password hash. · internal/adapter/http/profile_handler_test.go
- [x] **T017** Wire both use cases in `cmd/api/main.go` and run the verification gate: `go test ./...`, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, `sqlc generate` byte-stability. · cmd/api/main.go

## Dependencies & Execution Order

- **Phase 1** → **Phase 2** → **Phase 3** → **Phase 5** → **Phases 4, 6** → **Phase 7**.
- **Phase 4** shares `update_profile.go` with Phase 5, so they build in sequence.
- **Phase 6** depends on both routes existing.

## Phase 8: Convergence

- [ ] **T018** Add a test proving a refused password change leaves the stored password intact: record the hash, attempt the change through the use case with a wrong and with a missing current password, and confirm the old password still verifies and the new one does not, per SC-004 / FR-005 (partial)
- [ ] **T019** Add an `httptest` case posting `{}` to the update route and asserting `200` with every field unchanged, per spec: Edge Cases (partial)
