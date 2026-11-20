# Research: Admin User Listing (CU-AUTH-06 / HU-17)

## Decision 1: Optional filters in one statement

- **Decision**: each filter is a nullable `?` parameter, and the WHERE clause reads
  `(?3 IS NULL OR role = ?3)` and so on. The count query uses the identical predicate
  with the identical arguments.
- **Rationale**: the page and the total must agree. Two separately written WHERE
  clauses is exactly how a list ends up showing five rows and claiming a total of
  nine.
- **Alternatives considered**: building the SQL dynamically in Go; rejected because
  sqlc generates from static SQL and hand-built WHERE clauses invite injection.

## Decision 2: Case-insensitive search

- **Decision**: the search term is lowercased in the use case and matched with
  `lower(email) LIKE '%' || ? || '%'` and the same for `display_name`.
- **Rationale**: SQLite's `LIKE` is case-insensitive only for ASCII by default, and
  relying on that switch is a footgun. Lowercasing both sides makes the behaviour
  explicit and testable.
- **Alternatives considered**: `LIKE ... COLLATE NOCASE`; rejected because it is
  ASCII-only and hides the assumption.

## Decision 3: Stable ordering

- **Decision**: order by `created_at DESC, id ASC`.
- **Rationale**: two accounts registered in the same instant would otherwise be
  ordered arbitrarily, and a page boundary could repeat one and skip the other.
- **Alternatives considered**: `created_at DESC` alone; rejected as unstable.

## Decision 4: Where validation lives

- **Decision**: pagination bounds and filter values are validated in the domain, not
  the handler. The handler only parses strings into numbers and passes them on.
- **Rationale**: the rules are business rules, and a later non-HTTP caller of the use
  case must get the same guarantees.
- **Alternatives considered**: validating in the handler; rejected because it puts
  business rules at the transport edge.

## Decision 5: Role and level validation

- **Decision**: a role is validated against the existing `ParseRole`, so an unknown
  value is refused with a message naming the parameter. A learning level is compared
  against the stored text without a fixed scale, because registration does not
  constrain it.
- **Rationale**: refusing a nonsense role is useful; refusing a level the platform
  never constrained would block legitimate data.
- **Alternatives considered**: validating levels against the CEFR scale; rejected
  until registration enforces it.

## Resolved unknowns

None remain open for this brief.
