# Implementation Plan: User Logout (CU-AUTH-04 / HU-19)

## Summary

Add `POST /api/identity/auth/logout` behind two composable middlewares. One reads
the `Authorization: Bearer` credential, verifies it with the pinned-HS256 provider,
and puts the caller's user id and role in the request context. The other refuses a
caller whose role is not on a route's allow list. The handler ends only the session
named by the submitted refresh token, and only when that token belongs to the
caller, answering `204`. A missing, malformed, expired, or foreign token gets the
same `401 INVALID_TOKEN` refusal, so the endpoint never becomes a way to probe
other learners' sessions.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. This is the JWT and RBAC middleware the constitution places in `internal/adapter/http`; the contract is updated first. |
| II. Approved Stack and Idiomatic Go | PASS. `chi`, `jwt/v5`, and `sqlc` cover the work; no new dependency. |
| III. Phase-Scoped Go Skills | PASS. The security and REST skills were loaded before design. |
| IV. Secure Configuration and Verified Changes | PASS. The signing secret stays in configuration; the use case, adapter, middlewares, and handler are all tested. |
| V. Focused Changes and Dependency Discipline | PASS. Additive route and middleware; the existing schema already supports scoped revocation. |

No violations.

## Project Structure

- `internal/domain/errors.go` (read) - `ErrForbidden` and `ErrInvalidToken` already exist; no edit expected.
- `internal/adapter/http/middleware.go` (new) - `Authenticate` and `RequireRole`.
- `internal/adapter/http/logout_handler.go` (new) - the sign-out handler.
- `internal/adapter/http/router.go` (edit) - the protected route group.
- `internal/usecase/logout.go` (new) - the sign-out use case.
- `internal/usecase/ports.go` (edit) - `RevokeByJTI` on the refresh port.
- `internal/adapter/sqlite/refresh_token_store.go` (edit) - owner-scoped revocation.
- `db/queries/tokens.sql` (edit) - the scoped revoke query.
- `cmd/api/main.go` (edit) - wire the use case.
- `contracts/identity.openapi.yaml` and `shared/contracts/identity.openapi.yaml` - the logout path plus the bearer security scheme.
- Tests: `middleware_test.go`, `logout_handler_test.go`, `logout_test.go`, `logout_store_test.go`.

Structure Decision: middleware lives in the HTTP adapter because it knows about requests; the use case knows only about ownership and revocation.

## Key Decisions

See `research.md`. The load-bearing choice is that sign-out identifies the caller by
access token only, and ownership is decided by the refresh token's stored owner.
