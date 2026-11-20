# Research: User Login (CU-AUTH-02 / HU-02)

Phase 0 research for the login feature. No open question from the brief blocked
the design, so the entries below record the genuine choices the plan made while
resolving the feature's unknowns.

## Decision 1: Access token signing

- **Decision**: sign the access token with `github.com/golang-jwt/jwt/v5` using
  HMAC-SHA256 and a symmetric secret read from the `JWT_SECRET` configuration.
- **Rationale**: the constitution mandates `golang-jwt/jwt/v5` for short-lived
  access tokens and rotating refresh tokens; the service is the sole issuer, so a
  symmetric secret is the smallest sound choice.
- **Alternatives considered**: RSA with keys from configuration; rejected as
  disproportionate for a single issuer.

## Decision 2: Refresh token persistence and rotation

- **Decision**: persist each refresh token in a new `refresh_tokens` table keyed
  by a unique `jti`, storing a hash of the token, the user id, the expiry, and the
  revocation state; consuming a token revokes it and issues the successor.
- **Rationale**: rotation requires per-token lookup and revocation; a unique jti
  per token is the rotation key.
- **Alternatives considered**: storing the raw token; rejected (hash + jti).

## Decision 3: Login attempt limit shape

- **Decision**: reuse the per-visitor limiter with a policy constructor; the login
  policy allows 10 attempts per 60-second window and answers the 11th with `429`
  using the existing `rate_limited` error code.
- **Alternatives considered**: a dedicated limiter type; rejected (duplicate
  machinery).

## Decision 4: Error codes

- **Decision**: reuse the centralized error envelope codes (`invalid_credentials`,
  `rate_limited`) rather than adding new ones; the brief's `INVALID_CREDENTIALS` is
  the name of that existing generic credential failure.
- **Alternatives considered**: new dedicated codes; rejected (one envelope, one
  code per outcome).

## Decision 5: Token claims

- **Decision**: the access token carries `sub` (user id), `role` (user role), `iat`,
  and `exp` (now plus 15 minutes), signed HS256.

## Decision 6: Email validation at sign-in

- **Decision**: validate the normalized email with the domain validator before any
  credential verification or token issuance.

## Resolved unknowns

None remain open for the brief as written; every design choice above came from the
stack, constitution, and the literals the brief pinned.
