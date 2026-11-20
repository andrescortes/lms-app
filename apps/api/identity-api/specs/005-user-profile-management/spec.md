# Feature Specification: User Profile Management (CU-AUTH-05 / HU-18)

**Source**: CU-AUTH-05

**Created**: 2026-11-20

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - View My Account and Profile (Priority: P1)

A signed-in learner opens their profile page. The platform shows the details from
their account alongside the profile fields they filled in, so the learner can see
what the platform holds about them and which fields are still empty.

**Why this priority**: reading what exists is the foundation; nothing can be edited
until it can be shown.

**Independent Test**: A learner with a signed-in session asks for their profile and
receives their email, display name, role, and profile fields, with no password
information of any kind.

**Acceptance Scenarios**:

1. **Given** a signed-in learner, **When** they open their profile, **Then** the
   platform returns their id, email, display name, role, learning level, created
   date, and their profile fields (native language, learning goals, timezone,
   avatar URL), and never their password.
2. **Given** a learner who registered but never filled in a profile, **When** they
   open their profile, **Then** every profile field is empty rather than an error.

### User Story 2 - Change a Profile Field (Priority: P1)

A learner updates one thing about themselves, such as their learning goals or
timezone, without having to resend everything else. Fields they did not touch keep
their current value.

**Why this priority**: a form that demands every field at once is unusable in
practice; a partial update is what makes the screen work.

**Independent Test**: A learner changes only their goals, then reads the profile
back, and every other field is exactly what it was.

**Acceptance Scenarios**:

1. **Given** a learner with a filled profile, **When** they send a new learning
   goals value and nothing else, **Then** the goals change and every other field
   keeps its previous value.
2. **Given** a learner, **When** they send a new display name, **Then** the display
   name changes and the profile fields are untouched.

### User Story 3 - Change My Password Safely (Priority: P1)

A learner decides to change their password. The platform asks for the current
password, checks that it is right, checks that the new one meets the password rules,
and then keeps only the protected form of it.

**Why this priority**: a password change without re-proof of the current one lets
anyone who finds an open session take the account permanently.

**Independent Test**: A learner changes their password with the correct current
password and a compliant new one, then signs in with the new password.

**Acceptance Scenarios**:

1. **Given** a learner whose session is valid, **When** they send a new password
   with the correct current password, **Then** the password changes and the new one
   is the only one that works afterwards.
2. **Given** a learner sends a new password but no current password, **When** the
   request is processed, **Then** the platform refuses with `401` and
   `INVALID_CREDENTIALS` and the existing password still works.
3. **Given** a learner sends a wrong current password, **When** the request is
   processed, **Then** the platform refuses with `401` and the existing password
   still works.
4. **Given** a learner sends a new password that does not meet the password rules,
   **When** the request is processed, **Then** the platform refuses with `422` and
   the existing password still works.

### User Story 4 - Only I Can See and Change My Profile (Priority: P1)

A person tries to reach a profile without a valid session. The platform refuses
before any profile data is read or written.

**Why this priority**: profile data is personal; it must never be readable without
a proven identity.

**Independent Test**: A request to the profile routes without a bearer token, or
with an expired one, returns `401` and no profile data.

**Acceptance Scenarios**:

1. **Given** a request with no bearer credential, **When** it reaches the profile
   routes, **Then** the platform returns `401`.
2. **Given** a request with a credential whose lifetime has passed, **When** it
   reaches the profile routes, **Then** the platform returns `401`.

## Edge Cases

- A learner sends an empty body to the update route: every field stays as it was,
  and the platform still answers with the current profile.
- A learner sends `null` for a profile field: the field is cleared, not ignored.
  An absent field is left alone. These are different requests.
- A learner sends a display name of spaces only: it is refused as a validation
  failure rather than stored as blank.
- A learner sets a new password identical to the current one: it is accepted; the
  rules only constrain strength.
- A credential whose signing method the platform does not issue (for instance
  `alg=none`) is refused the same as an expired one.
- A profile read for an account that no longer exists returns `404`.

## Requirements

### Functional Requirements

- **FR-001**: The platform MUST return the signed-in learner's account details and
  profile fields on request, and MUST never include or echo the password in any
  form.
- **FR-002**: The platform MUST treat every profile field as optional: an unset
  field is reported as empty rather than as an error.
- **FR-003**: The platform MUST apply an update as a partial change: fields present
  in the request take their new value, fields absent keep their current value.
- **FR-004**: The platform MUST refuse an update whose display name is present but
  blank with a validation outcome, and MUST NOT store a blank display name.
- **FR-005**: The platform MUST refuse a password change that arrives without the
  current password, and MUST refuse one whose current password does not match. In
  both cases it returns `401` with `INVALID_CREDENTIALS` and leaves the stored
  password unchanged.
- **FR-006**: The platform MUST validate a new password against the password rules
  and refuse a weak one with `422`, leaving the stored password unchanged.
- **FR-007**: The platform MUST keep only the protected form of a new password and
  MUST update the account and the profile in one indivisible write.
- **FR-008**: Both profile routes MUST require a valid bearer credential and MUST
  refuse a missing, malformed, or expired one with `401`.

## Key Entities

- **Account**: The learner's identity. Carries id, email, display name, role,
  learning level, and creation date.
- **Profile**: The learner's personal record. Carries native language, learning
  goals, timezone, and avatar URL, any of which may be unset.
- **Credential**: The bearer proof of identity the caller presents; its lifetime
  and signing method decide whether it is accepted.

## Success Criteria

### Measurable Outcomes

- **SC-001**: 100% of profile responses carry no password material in any form,
  across every outcome including an error.
- **SC-002**: Every valid profile read returns `200` with the account and profile
  fields, and every unset profile field reads as empty.
- **SC-003**: A partial update changes exactly the fields it names; 100% of other
  fields keep their previous value.
- **SC-004**: 100% of refused password changes leave the stored password unchanged,
  proven by signing in with the old password afterwards.
- **SC-005**: 100% of requests without a valid bearer credential to the profile
  routes return `401` and read no profile data.

## Assumptions

- The learner edits only their own account; there is no admin path to another
  learner's profile in this slice.
- Email is not editable in this slice; changing it would need verification.
- Role and learning level are not editable by the learner.
- The profile page reads, then writes; there is no separate restore-defaults
  behaviour.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The read endpoint is `GET /api/identity/users/me`.
- The update endpoint is `PATCH /api/identity/users/me`.
- A successful read or update returns `200`.
- The body is `{"id", "email", "displayName", "role", "cefrLevel", "createdAt", "profile": {"nativeLanguage", "goals", "timezone", "avatarUrl"}}`.
- A missing account returns `404` with `NOT_FOUND`.
- A refused password change returns `401` with `INVALID_CREDENTIALS`.
- A weak password returns `422` with `WEAK_PASSWORD`.
