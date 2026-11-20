---
description: "Implementation tasks for Admin User Listing (CU-AUTH-06 / HU-17)"
---

# Tasks: Admin User Listing (CU-AUTH-06 / HU-17)

**Input**: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md), [research.md](research.md), [contracts/identity.openapi.yaml](contracts/identity.openapi.yaml).

**Paths**: relative to `apps/api/identity-api/`. Tests use stdlib `testing` and `httptest`.

## Phase 1: Setup

**Wave 1 — independent (different files):**

- [x] **T001** [P] Merge the `GET /api/identity/users` path and its schemas into `contracts/identity.openapi.yaml`, copy to `repo:shared/contracts/identity.openapi.yaml`, and confirm both parse with the path present. · contracts/identity.openapi.yaml, repo:shared/contracts/identity.openapi.yaml

**Checkpoint**: Canonical contract carries the list route.

## Phase 2: Foundational

**Wave 1 — blocking sequence:**

- [x] **T002** Add `ListUsersPage` and `CountUsers` to `db/queries/users.sql`: one shared nullable-filter predicate over role, cefr_level, lowercased email and display_name, ordered by `created_at DESC, id ASC`, selecting no password column. · db/queries/users.sql
- [x] **T003** Run `sqlc generate` and confirm byte-stability. · internal/adapter/sqlite/sqlc/
- [x] **T004** Add `internal/domain/listing.go`: `ValidatePage(is, limit)` for the 1..100 bound plus a `ListFilters` validator that parses a role and trims a search term. · internal/domain/listing.go
- [x] **T005** Add the `UserLister` port with `ListUsers(ctx, filter) (UserPage, error)` and the `UserSummary` read model. · internal/usecase/ports.go

**⟶ Wait for T005 to finish, then:**

- [x] **T006** Implement `internal/adapter/sqlite/user_lister.go`: `ListUsers` running the page query and the count query with identical arguments. · internal/adapter/sqlite/user_lister.go

**Checkpoint**: Storage and validation exist behind a port.

## Phase 3: User Story 1 — Browse All Learners (P1) 🎯 MVP

**Goal**: An administrator receives the first page newest first with the total.

**Independent Test**: Seed several accounts, list with no filters, and assert the order, the field set, and the total.

### Tests for User Story 1

- [x] **T007** [P] [US1] Add `internal/usecase/list_users_test.go` and `internal/adapter/sqlite/user_lister_test.go`: defaults applied, newest-first order, total equals the account count, and no password field in the read model. · internal/usecase/list_users_test.go, internal/adapter/sqlite/user_lister_test.go

### Implementation for User Story 1

- [x] **T008** [US1] Implement `internal/usecase/list_users.go`: normalize defaults (page 1, limit 20), validate, delegate. · internal/usecase/list_users.go
- [x] **T009** [US1] Implement `internal/adapter/http/user_list_handler.go` and register `GET /api/identity/users` inside an `ADMIN`-guarded group. · internal/adapter/http/user_list_handler.go, internal/adapter/http/router.go

**Checkpoint**: US1 works end to end; MVP complete.

## Phase 4: User Story 2 — Narrow the List (P1)

**Goal**: Role, level, and case-insensitive text filters narrow both the page and the total.

**Independent Test**: With mixed accounts, each filter returns only its matches and an empty result returns total zero.

### Tests for User Story 2

- [x] **T010** [P] [US2] Add filter tests at the store and use case: role only, level only, mixed-case search matching email and display name, combined filters, and an empty result with total zero. · internal/adapter/sqlite/user_lister_test.go, internal/usecase/list_users_test.go

### Implementation for User Story 2

- [x] **T011** [US2] Wire the filters through the use case and the adapter so the page and count share one argument set. · internal/usecase/list_users.go, internal/adapter/sqlite/user_lister.go

**Checkpoint**: US2 is independently verifiable.

## Phase 5: User Story 3 — Move Through Pages (P2)

**Goal**: Paging returns every account exactly once with stable ordering.

**Independent Test**: Page through the whole list and assert the union is every account, no duplicates, newest first.

- [x] **T012** [P] [US3] Add a paging test that walks the whole list in pages and asserts no account repeats or is skipped, plus a page-past-the-end case. · internal/adapter/sqlite/user_lister_test.go

**Checkpoint**: US3 is independently verifiable.

## Phase 6: User Story 4 — Only Administrators (P1)

**Goal**: `401` without a credential, `403` for a non-administrator.

- [x] **T013** [P] [US4] Add `httptest` cases asserting `401` with no bearer and `403 FORBIDDEN` for a STUDENT caller, with no account data in either body. · internal/adapter/http/user_list_handler_test.go

**Checkpoint**: US4 is independently verifiable.

## Phase 7: User Story 5 — Refuse Bad Parameters (P2)

**Goal**: Invalid page, limit, role, or level are refused naming the parameter.

- [x] **T014** [P] [US5] Add `internal/domain/listing_test.go` and handler cases for page 0, limit 0, limit 101, an unknown role, and a non-numeric page. · internal/domain/listing_test.go, internal/adapter/http/user_list_handler_test.go

**Checkpoint**: US5 is independently verifiable.

## Phase 8: Polish

- [x] **T015** Add the `200` handler case asserting the exact body shape and no password material, and wire `ListUsersUseCase` in `cmd/api/main.go`. · internal/adapter/http/user_list_handler_test.go, cmd/api/main.go
- [x] **T016** Run the verification gate: `go test ./...`, `golangci-lint run`, `CGO_ENABLED=0 go build ./cmd/api`, `sqlc generate` byte-stability, and confirm the contract copies match.

## Dependencies & Execution Order

- **Phase 1** → **Phase 2** → **Phase 3** → **Phases 4, 5** → **Phases 6, 7** → **Phase 8**.
- **Phase 4** and **Phase 5** touch the same two files and build in sequence.
