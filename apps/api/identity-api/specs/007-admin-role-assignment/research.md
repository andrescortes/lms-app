# Research: Admin User Role Assignment (CU-AUTH-07 / HU-17)

## Decision 1: Where the self-change guard lives

- **Decision**: the use case compares the actor id from the verified token with the
  target id and refuses before calling the repository.
- **Rationale**: the guard must hold for any caller, not only the HTTP route. Putting
  it in the use case means a future CLI or admin script inherits it.
- **Alternatives considered**: refusing in the handler; rejected because a second
  caller path would bypass it.

## Decision 2: Role validation

- **Decision**: reuse `domain.ParseRole`, whose accepted set is exactly the four
  stored role strings. A value outside it is refused with a validation error naming
  the role.
- **Rationale**: the database CHECK constraint and the domain parser already agree;
  a second parser would be a third source of truth.
- **Alternatives considered**: accepting a role regardless and letting SQLite's CHECK
  reject it; rejected because the failure surfaces as an opaque constraint error.

## Decision 3: Returning the updated account

- **Decision**: `UpdateUserRole` is a single `UPDATE ... RETURNING`, so the returned
  row is the post-change state and there is no window where another writer could
  change it between the write and the read.
- **Rationale**: a separate read after the write can return a different value under
  concurrency, which would make the response a lie.
- **Alternatives considered**: update then select; rejected.

## Decision 4: Unknown role versus malformed body

- **Decision**: a body that does not parse is `400`. A body that parses but carries
  an unknown role, or an unknown field, is `422` for the unknown role and `400` for
  the unknown field.
- **Rationale**: the two failures have different causes and different fixes.
- **Alternatives considered**: one code for both; rejected as unhelpful to a client.

## Decision 5: Case sensitivity of roles

- **Decision**: roles are compared exactly, so `admin` is not `ADMIN`.
- **Rationale**: the stored values are upper case and the CHECK constraint is exact;
  silently upper-casing would make the API accept something the rest of the system
  does not define.
- **Alternatives considered**: normalise the case; rejected.

## Resolved unknowns

None remain open for this brief.
