# Data Model: Session Refresh Rotation (CU-AUTH-03 / HU-03)

## Entities

### RefreshTokenRecord (existing table, no schema change)

- Fields: `jti` (primary key), `token_hash`, `user_id`, `expires_at`,
  `revoked_at` (nullable), `created_at`.
- Validation rules: `jti` unique; `token_hash` is the SHA-256 of the opaque token
  and never the token itself; `user_id` references a user and cascades on delete;
  `expires_at` is in the UTC text format the login slice writes.
- Relationships: many records per user. The set of records for one user with
  `revoked_at` null and `expires_at` in the future is that user's active chain.

### AccessToken (issued, not persisted)

- Claims: `sub` (user id), `role`, `iat`, `exp` = issue plus 15 minutes; HS256.

## Validation Rules

- A presented token is matched by `token_hash`. No row means the token was never
  issued.
- A matched row with `revoked_at` set means the token was already spent: this is
  reuse.
- A matched row with `expires_at` before now means the token expired. Expiry is
  refused but does not revoke anything else.
- A matched row that is active (unrevoked, unexpired) is the only case that
  rotates.

## State Transitions

| From | Event | To |
| --- | --- | --- |
| active | refresh | revoked, and a new active record is inserted for the same user |
| revoked | presented again | every active record for that user becomes revoked; the presented record stays revoked |
| expired | presented | unchanged; refusal only |
| absent | presented | unchanged; refusal only |

All transitions run inside one transaction. A failure writing the replacement
leaves the presented record unrevoked.
