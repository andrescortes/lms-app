# Implementation Plan: User Profile Management (CU-AUTH-05 / HU-18)

## Summary

Add `GET` and `PATCH /api/identity/users/me` inside the existing `Authenticate`
route group. The read joins the account with its profile and returns both, never
the password. The write is a partial update: fields present in the body change,
absent fields keep their value, and an explicit `null` clears a profile field. A
password change is allowed only when the caller supplies and proves their current
password, is validated against the existing password rules, and is stored only as
its protected hash. Account and profile writes share one transaction.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. Handlers delegate to use cases over ports; the contract is updated before the handlers; the routes reuse the existing middleware rather than adding a sibling dependency. |
| II. Approved Stack and Idiomatic Go | PASS. `chi`, `sqlc`, and `bcrypt` cover the work; no new dependency. |
| III. Phase-Scoped Go Skills | PASS. The security, database, REST, and design skills were loaded before design. |
| IV. Secure Configuration and Verified Changes | PASS. Passwords stay in the protected hash; the use cases and both adapters get tests. |
| V. Focused Changes and Dependency Discipline | PASS. Additive routes and queries on the existing schema. |

No violations.

## Project Structure

- `db/queries/users.sql` (edit) - joined read, account update, profile update.
- `internal/adapter/sqlite/profile_store.go` (new) - `GetUserProfile`, `UpdateUserProfile`.
- `internal/usecase/ports.go` (edit) - the profile store port.
- `internal/usecase/get_profile.go` (new) - the read use case.
- `internal/usecase/update_profile.go` (new) - the partial update use case.
- `internal/adapter/http/profile_handler.go` (new) - both handlers.
- `internal/adapter/http/router.go` (edit) - the two protected routes.
- `internal/adapter/http/respond_error.go` (edit) - the `WEAK_PASSWORD` code.
- `cmd/api/main.go` (edit) - wire both use cases.
- `contracts/identity.openapi.yaml` and `shared/contracts/identity.openapi.yaml` - both paths and schemas.
- Tests: `profile_store_test.go`, `get_profile_test.go`, `update_profile_test.go`, `profile_handler_test.go`.

Structure Decision: profile persistence gets its own adapter file so the user repository keeps its single responsibility of identity lookups.

## Key Decisions

See `research.md`. The load-bearing choice is how "absent" and "null" are told
apart in the partial update, which the request cannot express with plain Go strings.
