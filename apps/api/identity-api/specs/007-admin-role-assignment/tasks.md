---
description: "Implementation tasks for Admin User Role Assignment (CU-AUTH-07 / HU-17)"
---

# Tasks: Admin User Role Assignment (CU-AUTH-07 / HU-17)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: relative to `apps/api/identity-api/`. Tests use stdlib `testing` and `httptest`.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

**Wave 1 — independent (different files):**

- [x] **T001** [P] Merge the `PATCH /api/identity/users/{id}/role` path and its schemas into `contracts/identity.openapi.yaml`, copy to `repo:shared/contracts/identity.openapi.yaml`, and confirm both parse with the path present. · contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml

**Checkpoint**: Canonical contract carries the role route.

## Phase 2: Foundational

**Wave 1 — blocking sequence:**

- [x] **T002** Add `UpdateUserRole` to `db/queries/users.sql`: `UPDATE users SET role = ? WHERE id = ? RETURNING id, email, display_name, role, cefr_level, created_at`. · db/queries/users.sql
- [x] **T003** Run `sqlc generate` and confirm byte-stability. · internal/adapter/sqlite/sqlc/
- [x] **T004** Add `internal/domain/roles.go`: `ValidateAssignedRole(role)` using `ParseRole`, and `ActorMayChangeRole(actorID, targetID)` refusing a self-change. · internal/domain/roles.go
- [x] **T005** Add the `UserRoleUpdater` port with `UpdateUserRole(ctx, userID string, role domain.Role) (domain.User, error)`. · internal/usecase/ports.go

**⟶ Wait for T005 to finish, then:**

- [x] **T006** Implement `internal/adapter/sqlite/user_role_updater.go` over the returning query, mapping `sql.ErrNoRows` to `domain.ErrNotFound`. · internal/adapter/sqlite/user_role_updater.go

**Checkpoint**: Storage and validation exist behind a port.

## Phase 3: User Story 1 — Change Another Account's Role (P1) 🎯 MVP

**Goal**: An administrator changes another account's role and receives the updated account.

**Independent Test**: Change a learner to TEACHER and assert the response and the stored row both show TEACHER.

### Tests for User Story 1

- [x] **T007** [P] [US1] Add `internal/usecase/change_user_role_test.go` and `internal/adapter/sqlite/user_role_updater_test.go`: a valid change returns the updated user and persists; setting the existing role succeeds unchanged. · internal/usecase/change_user_role_test.go, internal/adapter/sqlite/user_role_updater_test.go

### Implementation for User Story 1

- [x] **T008** [US1] Implement `internal/usecase/change_user_role.go`: validate, guard, delegate, return the updated user. · internal/usecase/change_user_role.go
- [x] **T009** [US1] Implement `internal/adapter/http/user_role_handler.go` and register `PATCH /api/identity/users/{id}/role` in the ADMIN group. · internal/adapter/http/user_role_handler.go, internal/adapter/http/router.go

**Checkpoint**: US1 works end to end; MVP complete.

## Phase 4: User Story 2 — No Self-Change (P1)

**Goal**: An administrator targeting their own account is refused with 403.

- [x] **T010** [P] [US2] Add self-change tests at domain, use case, and handler: a same-id request returns `403 FORBIDDEN`, the repository is never called, and the role is unchanged. · internal/domain/roles_test.go, internal/usecase/change_user_role_test.go, internal/adapter/http/user_role_handler_test.go

**Checkpoint**: US2 independently verifiable.

## Phase 5: User Story 3 — Refuse an Unknown Role (P1)

**Goal**: A role outside the four defined values returns a validation outcome.

- [x] **T011** [P] [US3] Add role-validation tests: each of the four accepted, and `admin`, `WIZARD`, and empty refused with the repository untouched. · internal/domain/roles_test.go, internal/adapter/http/user_role_handler_test.go

**Checkpoint**: US3 independently verifiable.

## Phase 6: User Story 4 — Administrators Only (P1)

**Goal**: `401` without a credential, `403` for a non-administrator, no account touched.

- [x] **T012** [P] [US4] Add `httptest` cases: no bearer returns `401`, a STUDENT caller returns `403 FORBIDDEN`, and neither writes. · internal/adapter/http/user_role_handler_test.go

**Checkpoint**: US4 independently verifiable.

## Phase 7: User Story 5 — Unknown Account (P2)

**Goal**: A role change for a missing id returns `404`.

- [x] **T013** [P] [US5] Add the `404 NOT_FOUND` case through the handler and the adapter's not-found mapping. · internal/adapter/http/user_role_handler_test.go, internal/adapter/sqlite/user_role_updater_test.go

**Checkpoint**: US5 independently verifiable.

## Phase 8: Polish

- [x] **T014** Add the `200` handler case asserting the body shape with no password, and the `400` case for a malformed or unknown-field body. · internal/adapter/http/user_role_handler_test.go
- [x] **T015** Wire `ChangeUserRoleUseCase` in `cmd/api/main.go` and run the verification gate: `go test ./...`, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, `sqlc generate` byte-stability, contract copies identical. · cmd/api/main.go

## Dependencies & Execution Order

- **Phase 1** → **Phase 2** → **Phase 3** → **Phases 4, 5, 6, 7** → **Phase 8**.
- Phases 4 through 7 add tests to files Phase 3 creates, so they run after it, in order.
