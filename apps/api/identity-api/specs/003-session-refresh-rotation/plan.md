# Implementation Plan: Session Refresh Rotation (CU-AUTH-03 / HU-03)

## Summary

Add `POST /api/identity/auth/refresh`: the caller presents a refresh token, the
service revokes it, issues a replacement, and returns a fresh 15-minute access
token. A token that is malformed, expired, or unknown returns `401
INVALID_TOKEN`. A token that was already spent revokes the learner's whole chain
before refusing, so a stolen token cannot be replayed against another session.
The revoke-and-insert pair runs in one transaction so a crash cannot leave a
session half-rotated.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. The refresh handler delegates to a use case over ports; the contract is updated before the handler. |
| II. Approved Stack and Idiomatic Go | PASS. `jwt/v5`, `httprate`, `sqlc`, and `goose` cover the work with no new dependency. |
| III. Phase-Scoped Go Skills | PASS. The security, database, and REST skills were loaded before design. |
| IV. Secure Configuration and Verified Changes | PASS. The signing secret stays in configuration; the use case and both adapters get tests. |
| V. Focused Changes and Dependency Discipline | PASS. Additive endpoint on the existing schema; no migration needed. |

No violations.

## Project Structure

- `internal/adapter/security/jwt_token_provider.go` (edit) - add `Verify`.
- `internal/adapter/sqlite/refresh_token_store.go` (edit) - atomic rotation.
- `db/queries/tokens.sql` (edit) - rotation queries.
- `internal/usecase/refresh_token.go` (new) - refresh use case.
- `internal/usecase/ports.go` (edit) - the rotation port.
- `internal/adapter/http/refresh_handler.go` (new) - refresh handler.
- `internal/adapter/http/router.go` (edit) - register the route.
- `internal/adapter/http/respond_error.go` (edit) - `INVALID_TOKEN` mapping.
- `cmd/api/main.go` (edit) - wire the use case.
- `contracts/identity.openapi.yaml` and `shared/contracts/identity.openapi.yaml` - refresh path.
- Tests: `refresh_token_test.go`, `jwt_token_provider_test.go`, `refresh_store_test.go`, `refresh_handler_test.go`.

Structure Decision: the feature reuses the login slice's layers unchanged; rotation is a new use case over the same ports plus one adapter method.

## Key Decisions

See `research.md`; the load-bearing choice is that a spent token is recognised by keeping revoked rows rather than deleting them.
