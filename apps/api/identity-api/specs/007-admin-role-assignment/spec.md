# Feature Specification: Admin User Role Assignment (CU-AUTH-07 / HU-17)

**Source**: CU-AUTH-07

**Created**: 2026-11-20

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Change Another Account's Role (Priority: P1)

An administrator promotes a learner to teacher, or grants administrator access to
a colleague. They pick the account, choose a new role from the platform's four, and
the platform applies it and shows the updated account.

**Why this priority**: without this, every role change needs direct database work.
It is the whole point of the slice.

**Independent Test**: An administrator changes a learner's role to teacher and
receives the updated account showing the new role; listing that account afterwards
shows the change persists.

**Acceptance Scenarios**:

1. **Given** an administrator signed in and an existing learner account, **When**
   the administrator sets that account's role to teacher, **Then** the platform
   returns `200` with the account carrying the new role, and the change is stored.
2. **Given** an administrator signed in, **When** they set an account's role to the
   role it already has, **Then** the platform returns `200` and nothing about the
   account changes.

### User Story 2 - An Administrator Cannot Change Their Own Role (Priority: P1)

An administrator tries to change their own role, for example to demote themselves.
The platform refuses, so an administrator cannot accidentally lock themselves out
or escalate their own privileges.

**Why this priority**: this guard is the difference between a safe admin panel and
one that can strand the platform with no administrator.

**Independent Test**: An administrator targets their own account id and is refused
with `403`; their role is unchanged afterwards.

**Acceptance Scenarios**:

1. **Given** an administrator whose own account id is the target, **When** they
   request a role change, **Then** the platform returns `403` with the code
   `FORBIDDEN` and their role is unchanged.

### User Story 3 - Refuse a Role That Does Not Exist (Priority: P1)

An administrator (or a broken client) sends a role the platform does not have. The
platform refuses and names the parameter rather than storing nonsense.

**Why this priority**: the database constrains the role column, so a bad value that
reaches storage fails at the last possible moment with an opaque error.

**Independent Test**: Sending a role outside the four known values returns a
validation outcome and the target account's role is unchanged.

**Acceptance Scenarios**:

1. **Given** an administrator signed in, **When** they send a role the platform does
   not define, **Then** the platform returns a validation outcome naming the role
   and the account's role is unchanged.

### User Story 4 - Only Administrators Can Assign Roles (Priority: P1)

A learner, or someone without a session, tries to change a role. The platform
refuses before touching any account.

**Why this priority**: role assignment is privilege escalation by definition. Any
leak here lets a learner make themselves an administrator.

**Independent Test**: A request without a session returns `401`; a signed-in learner
returns `403`; neither changes any account.

**Acceptance Scenarios**:

1. **Given** a request with no valid bearer credential, **When** it reaches the role
   route, **Then** the platform returns `401` and no account changes.
2. **Given** a signed-in learner who is not an administrator, **When** they request
   a role change, **Then** the platform returns `403` with the code `FORBIDDEN` and
   no account changes.

### User Story 5 - A Missing Account Is Reported Clearly (Priority: P2)

An administrator targets an account id that no longer exists. The platform says the
account was not found rather than reporting success.

**Why this priority**: a silent success on a missing account hides a broken client
and makes the admin believe a change landed.

**Independent Test**: A role change against an unknown id returns `404`.

**Acceptance Scenarios**:

1. **Given** an administrator targets an id with no account, **When** the request is
   processed, **Then** the platform returns `404` with the code `NOT_FOUND`.

## Edge Cases

- The actor and target are compared by the account id from the verified token, never
  by anything in the request body.
- A request body that is not valid JSON returns `400`, not a role error.
- A body carrying fields other than the role is refused as a malformed request.
- A role sent in the wrong letter case is not the same role; the platform's roles are
  upper case and a lower-case value is refused.
- The response never includes a password in any form, on success or on error.

## Requirements

### Functional Requirements

- **FR-001**: The platform MUST set the target account's role to one of the four
  defined roles and MUST return the updated account carrying the identifier, email,
  display name, role, learning level, and creation date, and never a password.
- **FR-002**: The platform MUST refuse a role outside the four defined values with a
  validation outcome naming the role, leaving the target account unchanged.
- **FR-003**: The platform MUST refuse a request whose acting administrator is the
  target account, with `403` and the code `FORBIDDEN`, leaving the role unchanged.
- **FR-004**: The platform MUST require an authenticated caller whose role is
  administrator, returning `401` without a valid credential and `403` for a
  signed-in caller who is not an administrator, before changing any account.
- **FR-005**: The platform MUST report `404` with the code `NOT_FOUND` when the
  target account does not exist.
- **FR-006**: The platform MUST take the acting administrator's identity only from
  the verified access token, never from the request body or path.
- **FR-007**: The platform MUST refuse a malformed request body with `400`.

## Key Entities

- **Account role**: One of the four capability tiers. This feature changes exactly
  that one field on one account.
- **Acting administrator**: The caller whose verified token carries the administrator
  role. Their identity decides whether they may act and whether the target is
  themselves.

## Success Criteria

### Measurable Outcomes

- **SC-001**: An administrator changes another account's role in a single request and
  receives the updated account.
- **SC-002**: 100% of attempts by an administrator to change their own role are
  refused, and their role is unchanged afterwards.
- **SC-003**: 100% of non-administrators receive `401` or `403` and change no account.
- **SC-004**: Every refused request leaves the target account's role exactly as it
  was; no refusal is a partial write.
- **SC-005**: No response contains password material, on success or on error.

## Assumptions

- An administrator may assign the administrator role to another account; there is no
  separate super-administrator tier in this slice.
- Demoting an account is allowed as long as the actor is not the target.
- No audit trail is written by this slice; recording who changed what is a later
  concern.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The endpoint is `PATCH /api/identity/users/{id}/role`.
- The request body is `{"role": "..."}`.
- A successful change returns `200` with `{"id", "email", "displayName", "role", "cefrLevel", "createdAt"}`.
- The allowed roles are `STUDENT`, `TEACHER`, `ADMIN`, `REVIEWER`.
- A self-change returns `403` with `FORBIDDEN`.
- A missing account returns `404` with `NOT_FOUND`.
- An invalid role returns `422`; a malformed body returns `400`.
