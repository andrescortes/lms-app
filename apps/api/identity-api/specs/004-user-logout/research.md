# Research: User Logout (CU-AUTH-04 / HU-19)

## Decision 1: How the caller is identified

- **Decision**: the caller is identified solely by the `sub` and `role` claims of a
  verified access token. No header, body field, or query parameter is read for
  identity.
- **Rationale**: trusting any caller-supplied identity would let one learner act as
  another. The verified token is the only value the service can vouch for.
- **Alternatives considered**: taking the user id from the request body; rejected as
  trivially forgeable.

## Decision 2: Where ownership is enforced

- **Decision**: the refresh token's owner is read from its stored row, and the
  comparison happens in the adapter's scoped revoke inside one statement. A voice
  mismatch and a missing row both return `ErrInvalidToken`.
- **Rationale**: doing the check in SQL means there is no window between "read the
  owner" and "revoke" where the state could change, and it keeps the port small.
- **Alternatives considered**: an explicit `FindByToken` then `Revoke` in the use
  case; rejected because two round trips can disagree.

## Decision 3: Distinguishing "foreign" from "missing"

- **Decision**: both produce the same `401 INVALID_TOKEN`.
- **Rationale**: telling a caller that a token exists but belongs to someone else
  confirms that token is real. The spec's FR-003 requires the same refusal.
- **Alternatives considered**: `403` for the foreign case; rejected as an existence
  oracle.

## Decision 4: The expired and revoked cases

- **Decision**: an expired or already-revoked refresh token is refused with
  `INVALID_TOKEN` and no row is changed.
- **Rationale**: the session is already over; revoking again is a no-op that would
  report success for something that was not the caller's to end.
- **Alternatives considered**: treat a repeat sign-out as success (`204`); rejected
  because it makes the endpoint report success for a token it cannot authorise.

## Decision 5: Bearer parsing

- **Decision**: accept exactly one `Authorization` header whose value is
  `Bearer <token>` with a non-empty token and a case-insensitive scheme, and reject
  anything else as invalid before any verification.
- **Rationale**: a strict parser cannot be confused by a second header or a scheme
  the service does not issue.

## Decision 6: Role restriction

- **Decision**: `RequireRole` reads the identity the `Authenticate` middleware put
  in the context and compares it against the route's allow list; an empty allow
  list is a programming error and panics at construction, not at request time.
- **Rationale**: a misconfigured route should fail loudly during startup rather than
  silently allow everyone.

## Resolved unknowns

None remain open for this brief.
