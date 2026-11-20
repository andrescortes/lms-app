# Data Model: User Logout (CU-AUTH-04 / HU-19)

## Entities

### AccessToken

- Claims: `sub` (user id), `role`, `iat`, `exp` = issue plus 15 minutes; HS256.
- Validation: verified signature, pinned `HS256`, unexpired, and both `sub` and
  `role` present.

### AuthenticatedIdentity (new, request-scoped)

- Fields: `UserID`, `Role`.
- Built only by the `Authenticate` middleware from a verified access token and
  placed in the request context. Never persisted.

### RefreshTokenRecord (existing table, no schema change)

- Fields: `jti`, `token_hash`, `user_id`, `expires_at`, `revoked_at`, `created_at`.
- Validation at sign-out: the row must exist, be unrevoked, be unexpired, and its
  `user_id` must equal the caller's identity.

## State Transitions

| From | Event | To |
| --- | --- | --- |
| active, owned by caller | sign-out | revoked |
| active, owned by another | sign-out | unchanged; refusal |
| revoked | sign-out | unchanged; refusal |
| expired | sign-out | unchanged; refusal |
| absent | sign-out | nothing to change; refusal |

The refusal cases return the same `401 INVALID_TOKEN` so a caller cannot tell a
foreign token from a missing one.
