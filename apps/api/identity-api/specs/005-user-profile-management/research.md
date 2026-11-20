# Research: User Profile Management (CU-AUTH-05 / HU-18)

## Decision 1: Telling "absent" apart from "null" in a partial update

- **Decision**: the request DTO carries each profile field and each account field as
  a pointer (`*string`). A field that is absent from the JSON leaves its pointer
  nil and is not touched. A field sent as `null` produces a non-nil marker that
  clears the stored value. Plain JSON decoding cannot distinguish these with a bare
  `string`, and the spec's edge cases require the difference.
- **Rationale**: without the distinction, a learner could never clear a learning
  goal, and a client that sends only `goals` would silently wipe the timezone.
- **Alternatives considered**: `omitempty` with plain strings; rejected because it
  cannot express "set to empty" separately from "leave alone".

## Decision 2: Where the password-change rules live

- **Decision**: the use case owns the sequence: require the current password when a
  new one is present, verify the current password through `PasswordVerifier`, then
  validate the new one with `domain.ValidatePasswordPolicy`, then hash through
  `PasswordHasher`. The handler never sees a hash.
- **Rationale**: these are business rules, and the domain and ports already exist;
  the adapter only stores what it is given.
- **Alternatives considered**: verify the password in the handler; rejected because
  it puts a business rule in the transport.

## Decision 3: Order of validation inside the password change

- **Decision**: check that the current password is present and correct before
  validating the new password's strength.
- **Rationale**: a caller who cannot prove identity should not learn whether their
  chosen new password would have been strong enough. It also matches FR-005's
  `401` taking precedence over FR-006's `422`.
- **Alternatives considered**: validate strength first, returning `422` to an
  unauthenticated caller; rejected as a strength oracle for a session the caller
  cannot prove they own.

## Decision 4: Atomicity of the update

- **Decision**: `UpdateUserProfile` takes one `*sql.Tx` covering the account row and
  the profile row, and rolls back both if either fails.
- **Rationale**: a password change that commits while its display-name change fails
  leaves the learner with a confusing half-applied state.
- **Alternatives considered**: separate transactions per table; rejected.

## Decision 5: Blank display name

- **Decision**: a display name that is present but empty or whitespace-only is
  refused with the existing validation outcome before any write.
- **Rationale**: the registration slice already guarantees a non-empty display name;
  allowing a blank one through the update route would break that invariant.
- **Alternatives considered**: fall back to the email local part as registration
  does; rejected because a learner clearing the field meant to clear it, and a
  silent substitution hides the mistake.

## Resolved unknowns

None remain open for this brief.
