# Data Model: User Login (CU-AUTH-02 / HU-02)

## Entities

### User (existing, read by login)

- Fields: id, normalized email, protected password representation, display name,
  role, learning level, created at.
- Relationships: one user has many refresh tokens; login reads the user by
  normalized email.

### RefreshToken (new)

- Fields: `jti` (primary key, unique per token), `tokenHash`, `userId`, `expiresAt`,
  `revokedAt`, `createdAt`.
- Validation rules: `jti` is unique and non-empty; `tokenHash` is non-empty; `userId`
  references a user; `expiresAt` is set at issue and in the future.
- Relationships: many refresh tokens to one user; a user has many refresh tokens.

### AccessToken (issued, not persisted)

- Fields/claims: `sub` (user id), `role` (user role), `iat`, `exp` (now plus 15
  minutes); signed HS256; TTL `15m`.

## Validation Rules

- The refresh token's expiry is evaluated at use; an expired or revoked token is
  rejected and never returns credentials.
- The access token's `exp` is evaluated at use; a token whose `exp` passed is
  rejected.

## State Transitions

- Login success: insert a refresh token (`expiresAt` = issue time plus `7d`,
  `revokedAt` unset), then return the access and refresh tokens.
- Failed login: no refresh token is written and no token is returned.
- Token rotation (later step): revoke the presented refresh token and issue the
  successor for the same user.
