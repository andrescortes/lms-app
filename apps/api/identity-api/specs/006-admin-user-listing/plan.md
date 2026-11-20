# Implementation Plan: Admin User Listing (CU-AUTH-06 / HU-17)

## Summary

Add `GET /api/identity/users`, restricted to administrators by the existing
`Authenticate` and `RequireRole("ADMIN")` middlewares. The endpoint returns one page
of account summaries newest first, with the total count of accounts matching the
filters. Filters are role, learning level, and a case-insensitive fragment matched
against email or display name. Invalid filters or pagination are refused with a
validation outcome naming the parameter rather than silently defaulted.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. The handler delegates to a use case over a consumer-defined port; the contract is updated first; the route reuses the existing middlewares. |
| II. Approved Stack and Idiomatic Go | PASS. `chi`, `sqlc`, and the standard library cover the work; no new dependency. |
| III. Phase-Scoped Go Skills | PASS. The database and REST skills were loaded before design. |
| IV. Secure Configuration and Verified Changes | PASS. The list selects no password column; the use case and adapter get tests. |
| V. Focused Changes and Dependency Discipline | PASS. Additive route and queries on the existing table. |

No violations.

## Project Structure

- `internal/domain/listing.go` (new) - filter and pagination validation.
- `db/queries/users.sql` (edit) - the paginated list and count queries.
- `internal/adapter/sqlite/user_lister.go` (new) - `ListUsers` over those queries.
- `internal/usecase/list_users.go` (new) - defaults, validation, delegation.
- `internal/usecase/ports.go` (edit) - the `UserLister` port.
- `internal/adapter/http/user_list_handler.go` (new) - query parsing and response.
- `internal/adapter/http/router.go` (edit) - the admin route.
- `cmd/api/main.go` (edit) - wire the use case.
- `contracts/identity.openapi.yaml` and `shared/contracts/identity.openapi.yaml` - the list path.
- Tests: `listing_test.go`, `user_lister_test.go`, `list_users_test.go`, `user_list_handler_test.go`.

Structure Decision: the list gets its own adapter file beside the profile store, so the identity lookup repository keeps one job.

## Key Decisions

See `research.md`. The load-bearing one is how the optional filters are expressed in
one SQL statement so the page and the count can never disagree.
