# Implementation Plan: User Login (CU-AUTH-02 / HU-02)

## Summary

Add `POST /api/identity/auth/login` to the Identity service: the handler
normalizes the submitted email, verifies the password against the stored protected
hash, and on valid credentials returns a `15m` access token plus a rotating `7d`
refresh token persisted with a unique jti; on invalid credentials it returns `401`
with the generic `INVALID_CREDENTIALS` code; and attempts are throttled at 10 per
minute per IP with `429`.

## Constitution Check

| Principle | Assessment |
| --- | --- |
| I. Inward-Only Dependencies and Contract-First HTTP | PASS. The handler depends on the login use case port; the login contract is authored before the handler; adapters implement the ports the use case owns. |
| II. Approved Stack and Idiomatic Go | PASS. Go, chi, sqlc, goose, bcrypt, and `github.com/golang-jwt/jwt/v5` (mandated by the constitution) cover the feature; domain and use case layers stay standard-library only. |
| III. Phase-Scoped Go Skills | PASS. The Go security, design-pattern, REST, and database skills were loaded before design and task breakdown. |
| IV. Secure Configuration and Verified Changes | PASS. The JWT signing secret comes from configuration; every use case gets unit tests and every SQLite adapter gets integration tests; the verification gates apply. |
| V. Focused Changes and Dependency Discipline | PASS. Additive endpoint plus its ports, adapters, migration, and contract; no unrelated files touched. |

No constitutional violations.

## Project Structure

Feature files:

- `internal/adapter/http/login_handler.go` (new) - login handler: decode, delegate, respond.
- `internal/adapter/http/router.go` (edit) - register the login route.
- `internal/adapter/http/visitor_limit.go` (edit) - login attempt-limit policy.
- `internal/adapter/security/jwt_token_provider.go` (new) - TokenProvider implementation.
- `internal/adapter/security/jwt_token_provider_test.go` (new) - token tests.
- `internal/adapter/sqlite/refresh_token_store.go` (new) - refresh token repository.
- `db/migrations/00003_refresh_tokens.sql` (new) - refresh token table.
- `db/queries/tokens.sql` (new) - refresh token queries.
- `internal/usecase/login_user.go` (new) - login use case.
- `internal/usecase/login_user_test.go` (new) - use case tests.
- `cmd/api/main.go` (edit) - wire login use case, token provider, refresh store, limiter.
- `.env.example` (edit) - document the JWT signing secret.
- `contracts/` and `shared/contracts/` - login endpoint contract.

Structure Decision: the feature attaches inside the existing inward-only hexagon; new behavior lands in the adapter and use case layers the constitution already reserved, plus one migration and the OpenAPI contract, with no layer crossed.
