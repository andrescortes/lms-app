# Feature Specification: Session Refresh Rotation (CU-AUTH-03 / HU-03)

**Source**: CU-AUTH-03

**Created**: 2026-11-20

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Refresh a Session with a Valid Token (Priority: P1)

A learner whose short-lived access token has expired presents their refresh token
and receives a new access token and a new refresh token. The token they presented
stops working, so a stolen refresh token is only useful once.

**Why this priority**: Without refresh, every learner is signed out every fifteen
minutes. This is the slice that makes a session last a working day.

**Independent Test**: A learner with a valid refresh token calls the refresh
endpoint and receives a working access token and a different refresh token; the
token they just used is rejected on the next attempt.

**Acceptance Scenarios**:

1. **Given** a learner holds an unexpired, unused refresh token, **When** the
   learner presents it to refresh, **Then** the system returns a new access token
   that lives 15 minutes with the user id and role, and a new refresh token that
   lives 7 days and is persisted.
2. **Given** a learner has refreshed once, **When** the learner presents the token
   used in the first refresh again, **Then** the system rejects that reuse as an
   invalid token.

### User Story 2 - Reject a Token That Is Invalid, Expired, or Revoked (Priority: P2)

A learner presents a refresh token that has expired, was already spent, or was
never issued by the platform. The system refuses and does not grant a new session.

**Why this priority**: A rotation flow is only worth having if the old token
stops working. Refusal is the half that makes rotation meaningful.

**Independent Test**: A learner presents an expired token, a malformed token, and
a token they already spent; each is refused with the same invalid-token outcome,
and no new token is returned.

**Acceptance Scenarios**:

1. **Given** a refresh token whose lifetime has passed, **When** the learner
   presents it, **Then** the system refuses with `401` and the code
   `INVALID_TOKEN`, and returns no tokens.
2. **Given** a refresh token that is malformed or was never issued, **When** the
   learner presents it, **Then** the system refuses with `401` and the code
   `INVALID_TOKEN`.

### User Story 3 - Revoke a Whole Chain When a Spent Token Is Replayed (Priority: P2)

A learner's refresh token has been used once already, and someone presents it a
second time. Because that token cannot legitimately be presented twice, the
platform treats it as a sign that the learner's session may be compromised: it
ends every session that learner holds, not just the one in question.

**Why this priority**: Reuse is the only signal the platform has that a token
leaked. Failing to act on it leaves the attacker with a working session.

**Independent Test**: A learner signs in twice, refreshing once; the operator
replays the spent token; both of the learner's sessions are refused afterwards.

**Acceptance Scenarios**:

1. **Given** a learner holds several active refresh tokens and one of them has
   been spent, **When** the spent token is presented again, **Then** the system
   refuses with `401` and the code `INVALID_TOKEN`, and every other active token
   for that learner is revoked as well.

### User Story 4 - Rate Limit Refresh Attempts (Priority: P3)

A visitor floods the refresh endpoint. The system caps the attempt rate and
answers later requests with a rate-limit outcome.

**Why this priority**: Same protection the sign-in endpoint already has, applied
to a second unauthenticated entry point.

**Independent Test**: A visitor exceeding the allowed attempts within a minute
receives a `429` response.

**Acceptance Scenarios**:

1. **Given** a visitor makes repeated refresh attempts, **When** the visitor
   exceeds 10 attempts in a minute, **Then** the system returns `429`.

## Edge Cases

- A refresh token presented twice in quick succession: the second attempt is
  treated as reuse, so the learner is signed out everywhere.
- A learner with several active sessions refreshing from one of them: the other
  sessions are untouched, unless a spent token is replayed.
- The access token issued by refresh carries the user id and the role, so
  authorisation downstream works the same as after sign-in.
- A token that is expired is refused without revoking anything else. Expiry is
  not suspicious; reuse is.
- The refresh endpoint reads no email or password, so a refusal never names an
  account.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST accept a refresh request carrying a refresh token
  and MUST return a 200 response with a new access token and a new refresh token
  when that token is valid, unused, and unexpired. The access token MUST live
  `15m` and carry the user id and role. The new refresh token MUST live `7d` and
  MUST be persisted with its own unique identifier.
- **FR-002**: The system MUST revoke a refresh token at the moment it is used to
  refresh, so the same token cannot be used a second time.
- **FR-003**: The system MUST return a 401 response with the code `INVALID_TOKEN`
  when the presented refresh token is malformed, was never issued, has expired,
  or has already been revoked, and MUST return no tokens in that case.
- **FR-004**: When a refresh token that has already been revoked is presented,
  the system MUST revoke every active refresh token belonging to that same user
  before returning `INVALID_TOKEN`, so a leaked token cannot be used against the
  learner's other sessions.
- **FR-005**: The revocation-and-issue of FR-001 and FR-002 MUST happen as one
  indivisible step: either the presented token is revoked and the replacement is
  persisted, or neither change is kept.
- **FR-006**: The system MUST allow at most 10 refresh attempts per minute per
  visitor and MUST return a 429 response when the limit is exceeded.

## Key Entities

- **Refresh token record**: The stored trace of one issued refresh token. Carries
  its own identifier, the learner it belongs to, when it expires, and when it was
  revoked.
- **Access token**: A short-lived credential issued by refresh. Carries the user
  id and role, and expires after `15m`.
- **Token chain**: Every refresh token currently belonging to one learner. A
  replay of a spent token revokes the whole chain.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A learner with a valid refresh token receives a new access token and
  a new refresh token in a single request.
- **SC-002**: A refresh token that has been used once is refused on every later
  attempt, 100% of the time.
- **SC-003**: Replaying a spent token leaves the learner with no usable refresh
  token; a follow-up refresh with any previous token is refused.
- **SC-004**: Expired, malformed, and unissued tokens all produce the same 401
  outcome with the code `INVALID_TOKEN`.
- **SC-005**: A visitor exceeding 10 refresh attempts in a minute receives a 429
  response.

## Assumptions

- The refresh endpoint is public, like sign-in; the token itself is the
  credential.
- Access tokens remain the caller's to present elsewhere; this feature issues
  them but does not verify them on other routes.
- A revoked token is kept rather than deleted, so a replay can be recognised
  instead of mistaken for an unissued token.
- The learner's client replaces both tokens after each successful refresh.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The endpoint is `POST /api/identity/auth/refresh`.
- A successful refresh returns `200 OK`.
- The request body is `{"refreshToken": "..."}`.
- The success body is `{"accessToken": "...", "refreshToken": "..."}`.
- The access token lives `15m`; the refresh token lives `7d`.
- A refused refresh returns `401` with the code `INVALID_TOKEN`.
- Exceeding the attempt limit returns `429`; the limit is 10 attempts/min/IP.
