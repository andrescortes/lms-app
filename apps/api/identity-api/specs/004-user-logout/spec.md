# Feature Specification: User Logout (CU-AUTH-04 / HU-19)

**Source**: CU-AUTH-04

**Created**: 2026-11-20

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Sign Out and End the Session (Priority: P1)

A learner who is signed in decides to leave the shared device. They ask the
platform to sign them out, presenting their refresh token and the access token
that proves who they are. The platform ends that session, so the refresh token
cannot be used to obtain a new one.

**Why this priority**: Without sign-out, a learner on a shared or borrowed device
has no way to stop someone else from continuing their session. It is the feature's
whole value.

**Independent Test**: A learner with a valid access token and refresh token signs
out, receives confirmation that nothing is returned in the response, and the
refresh token they used is refused if they try to refresh with it.

**Acceptance Scenarios**:

1. **Given** a signed-in learner with a valid access token and their refresh
   token, **When** the learner signs out, **Then** the platform ends that session
   and returns a `204 No Content` response.
2. **Given** a learner has signed out, **When** the learner presents the refresh
   token used at sign-out to refresh, **Then** the session refresh endpoint
   refuses it.

### User Story 2 - Only the Session Owner Can End a Session (Priority: P1)

A caller presents a valid access token for one account but a refresh token that
belongs to another. The platform refuses, so one learner cannot sign another out.

**Why this priority**: The endpoint ends a session, which is a destructive act. It
must be tied to the caller's own identity or it becomes a way to attack other
learners.

**Independent Test**: A learner signs out with their own access token and someone
else's refresh token, and is refused with `401` and the code `INVALID_TOKEN`;
the other learner's session still works.

**Acceptance Scenarios**:

1. **Given** a caller with a valid access token for one account and a refresh
   token belonging to a different account, **When** the caller requests sign-out,
   **Then** the platform refuses with `401` and the code `INVALID_TOKEN`, and the
   other account's session is unchanged.

### User Story 3 - Protected Routes Require a Valid Identity (Priority: P2)

A caller reaches a route that only signed-in learners may use, without a token, with
a malformed token, or with an expired one. The platform refuses with a generic
unauthenticated outcome instead of the route's own response.

**Why this priority**: Sign-out itself needs this gate. It also becomes the
pattern every later protected route reuses, so it is built and tested here.

**Independent Test**: Requests to the sign-out endpoint with no bearer token, a
malformed token, and an expired token each return `401` and never reach the
sign-out action.

**Acceptance Scenarios**:

1. **Given** a request without an `Authorization: Bearer` credential, **When**
   it reaches a protected route, **Then** the platform returns `401` and does not
   perform the protected action.
2. **Given** a request whose bearer token is malformed or whose lifetime has
   passed, **When** it reaches a protected route, **Then** the platform returns
   `401` and does not perform the protected action.

### User Story 4 - Restricted Routes Refuse the Wrong Role (Priority: P3)

A signed-in learner reaches a route reserved for a different role, such as a
teacher-only or administrator-only action. The platform refuses because their
identity is valid but their role is not allowed.

**Why this priority**: No route in this feature needs role restriction yet, but
the refusal behaviour is part of the same access-control pattern and is testable
on its own.

**Independent Test**: An authenticated caller whose role is not on a route's
allowed list receives `403`.

**Acceptance Scenarios**:

1. **Given** a signed-in caller with a valid identity whose role is not allowed on
   a route, **When** the caller requests that route, **Then** the platform returns
   `403` and does not perform the action.

## Edge Cases

- A refresh token that was already revoked (for instance by a refresh) presented at
  sign-out is refused; it is not owned as an active session any more.
- A refresh token that has expired is refused at sign-out.
- A malformed request body returns `400`, and the session is left untouched.
- An empty refresh token is a malformed request, not an authentication failure.
- A caller cannot sign another learner out, whether the other token exists or not;
  both cases produce the same refusal.
- Sign-out ends only the session named by the presented token. Other sessions for
  the same learner stay signed in.
- A bearer token that is present but uses an unexpected signing method is refused
  the same as an expired one.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST accept a sign-out request carrying a refresh token
  from a caller who presents a valid access token, MUST end the session that
  refresh token names, and MUST return a `204 No Content` response with no body.
- **FR-002**: The system MUST refuse a sign-out request whose refresh token has
  already been revoked or has expired, returning `401` with the code
  `INVALID_TOKEN` and leaving every session untouched.
- **FR-003**: The system MUST refuse a sign-out request whose refresh token belongs
  to a different account than the caller's access token, returning `401` with the
  code `INVALID_TOKEN`, and MUST leave the other account's session unchanged.
  A token that does not exist MUST produce the same refusal.
- **FR-004**: The system MUST require a valid access token on protected routes: a
  request with no bearer credential, a malformed token, an expired token, or a
  token signed with an unexpected method MUST be refused with `401` before the
  protected action runs.
- **FR-005**: The system MUST identify the caller from the access token it
  validated, exposing the caller's user id and role to protected handlers, and MUST
  NOT trust any caller-supplied identity other than that token.
- **FR-006**: The system MUST support restricting a route to a set of allowed
  roles and MUST refuse a signed-in caller whose role is not allowed with `403`
  and the code `FORBIDDEN`.
- **FR-007**: Sign-out MUST end only the session named by the presented refresh
  token; the learner's other active sessions MUST remain valid.

## Key Entities

- **Access token**: The short-lived credential a caller presents as proof of
  identity. Carries the user id and role and expires after `15m`.
- **Refresh token record**: The stored trace of one session. Carries its owner,
  expiry, and whether it was revoked. Legitimate sign-out revokes one record.
- **Authenticated identity**: The user id and role derived from a validated access
  token, made available to the code handling a protected route.
- **Role**: The capability tier carried by the access token. Route access may be
  restricted to a set of roles.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A signed-in learner who signs out receives `204 No Content` with an
  empty body in a single request.
- **SC-002**: A refresh token used at sign-out is refused by the refresh endpoint
  on every later attempt, 100% of the time.
- **SC-003**: A caller with a valid access token and another account's refresh
  token is refused, and the other account's session still refreshes successfully.
- **SC-004**: 100% of requests to a protected route without a valid access token
  return `401` and perform no protected action.
- **SC-005**: Sign-out ends exactly one session; a learner's other sessions
  refresh successfully after a sign-out.

## Assumptions

- The access token is the credential a caller presents to sign out; a caller
  cannot sign out by refresh token alone.
- Sign-out is idempotent from a caller's point of view only in the sense that the
  second attempt is refused rather than repeated successfully; a revoked token is
  not re-accepted.
- Access tokens are not individually revocable in this feature; a signed-out
  learner's access token keeps working until it expires. The refresh token, which
  would grant a new session, is ended. Revoking access tokens early is a later
  concern.
- Routes that need role restriction arrive with later features; this feature
  supplies the mechanism and proves it with tests.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The endpoint is `POST /api/identity/auth/logout`.
- A successful sign-out returns `204 No Content`.
- The request body is `{"refreshToken": "..."}`.
- The bearer credential is sent as `Authorization: Bearer <token>`.
- A refused sign-out for a bad or foreign token returns `401` with the code `INVALID_TOKEN`.
- A wrong-role refusal returns `403` with the code `FORBIDDEN`.
- The access token signing method is `HS256`.
