# Implementation Plan: Admin User Role Assignment (CU-AUTH-07 / HU-17)

## Summary

Add `PATCH /api/identity/users/{id}/role`, reachable only by an administrator. The
request names the account and the new role; the platform validates the role against
the four it defines, refuses a change to the caller's own account, and applies the
change in one statement that returns the updated account. Every refusal happens
before any write, so a rejected request leaves the account exactly as it was.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. The handler delegates to a use case over a consumer-defined port; the contract is updated first; the route reuses the existing middlewares. |
| II. Approved Stack and Idiomatic Go | PASS. `chi`, `sqlc`, and the domain role parser cover the work; no new dependency. |
| III. Phase-Scoped Go Skills | PASS. The REST and security skills were loaded before design. |
| IV. Secure Configuration and Verified Changes | PASS. The update returns no password column; the use case and adapter get tests. |
| V. Focused Changes and Dependency Discipline | PASS. Additive route and one query on the existing table. |

No violations.

## Project Structure

- `internal/domain/roles.go` (new) - role assignment validation and the self-change rule.
- `db/queries/users.sql` (edit) - `UpdateUserRole`.
- `internal/adapter/sqlite/user_role_updater.go` (new) - the update adapter.
- `internal/usecase/change_user_role.go` (new) - the use case.
- `internal/usecase/ports.go` (edit) - the `UserRoleUpdater` port.
- `internal/adapter/http/user_role_handler.go` (new) - the handler.
- `internal/adapter/http/router.go` (edit) - the admin route with a path parameter.
- `cmd/api/main.go` (edit) - wire the use case.
- `contracts/identity.openapi.yaml` and `shared/contracts/identity.openapi.yaml` - the role path.
- Tests: `roles_test.go`, `user_role_updater_test.go`, `change_user_role_test.go`, `user_role_handler_test.go`.

Structure Decision: the role update gets its own adapter file, keeping the list and
lookup repositories single-purpose.

## Key Decisions

See `research.md`. The load-bearing one is that the actor check uses the token
identity and runs before the repository call, so no refusal can be a partial write.
