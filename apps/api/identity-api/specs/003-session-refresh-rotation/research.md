# Research: Session Refresh Rotation (CU-AUTH-03 / HU-03)

## Decision 1: How a spent token is recognised

- **Decision**: revoked rows stay in `refresh_tokens` with `revoked_at` set rather
  than being deleted, and the rotation query distinguishes "no row" from "row with
  `revoked_at` set".
- **Rationale**: reuse detection only works if a spent token can be told apart
  from one that never existed. Deleting on revoke makes both look unknown, and the
  chain revocation in FR-004 would never fire.
- **Alternatives considered**: delete on revoke; rejected because it destroys the
  signal.

## Decision 2: Where the atomicity lives

- **Decision**: the transaction sits in one repository method,
  `Rotate(ctx, presentedToken, newToken, newExpiry)`, which reads the row, decides,
  and either revokes-and-inserts or revokes-the-chain, all inside `*sql.Tx`.
- **Rationale**: the use case should not orchestrate a transaction across several
  repository calls; the adapter owns persistence and SQLite has one writer.
- **Alternatives considered**: a `TxManager` port and several calls; rejected as
  more moving parts for no gain.

## Decision 3: Which token the client presents

- **Decision**: the refresh token is the opaque random string issued at login, not
  a JWT. The provider's new `Verify` exists for access-token verification and is
  not on the refresh path.
- **Rationale**: the login slice already issues an opaque token and stores its
  SHA-256 hash; rotation looks it up by hash. FR-002's "verify" is the lookup plus
  expiry and revocation checks.
- **Alternatives considered**: signing the refresh token as a JWT; rejected as it
  changes the login slice's contract for no security gain.

## Decision 4: Expiry evaluation

- **Decision**: expiry is compared in the adapter against the same UTC text
  format the schema stores, so no clock skew exists between writer and reader.
- **Rationale**: the login slice writes `expires_at` with `formatUTC`; comparing
  in the same format avoids parsing surprises.

## Decision 5: Rate limiting

- **Decision**: reuse `loginRateLimiter`'s `httprate.LimitBy` shape with the same
  peer-address key and a separate bucket per route.
- **Rationale**: httprate buckets are per middleware instance, so a shared limit is
  a construction choice; a separate instance keeps refresh attempts from spending
  the sign-in budget.

## Resolved unknowns

None remain open for this brief.
