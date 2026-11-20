# Feature Specification: User Login (CU-AUTH-02 / HU-02)

**Source**: CU-AUTH-02

**Created**: 2026-11-19

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Sign In with Valid Credentials (Priority: P1)

A learner who already registered an account opens the sign-in screen, submits the
email and password that belong to that account, and the system recognizes the
learner. The system then returns a short-lived access token and a longer-lived
refresh token so the learner can stay signed in.

**Why this priority**: Signing in is the gate every registered learner passes
before reaching learning content, so the successful path carries the feature.

**Independent Test**: A learner with a registered account submits the correct
email and password and receives an access token and a refresh token the platform
accepts.

**Acceptance Scenarios**:

1. **Given** a learner with a registered account, **When** the learner submits the
   registered email and the matching password, **Then** the system returns a 200
   response with an access token that lives `15m` and carries the user id and the
   learner role, plus a refresh token that lives `7d` and is persisted with a unique
   identifier.

### User Story 2 - Sign In with Invalid Credentials (Priority: P2)

A learner submits an email and password that do not match any account, or an
email that is not registered at all. The system rejects the attempt with a generic
failure and does not disclose whether an account exists for the email.

**Why this priority**: Failed sign-ins are part of the same journey, but the
successful path is the value-bearing slice, so the rejection story is second.

**Independent Test**: A learner submits an unknown email, or a known email with a
wrong password, and observes a single failure outcome that names no account
details.

**Acceptance Scenarios**:

1. **Given** a learner with a registered account, **When** the learner submits a
   wrong password, **Then** the system returns a 401 response with the generic code
   `INVALID_CREDENTIALS`.
2. **Given** a visitor who has never registered, **When** the visitor submits an
   email that is not registered, **Then** the system returns the same 401 response
   with the generic code `INVALID_CREDENTIALS`.

### User Story 3 - Rate Limit Sign In Attempts (Priority: P2)

A visitor repeatedly attempts to sign in, for example against a known account or
guessing credentials. The system stops the flood of attempts and answers later ones
with a rate-limit outcome.

**Why this priority**: Brute-force protection is a safety requirement of the
sign-in flow and stands on its own as a testable behavior.

**Independent Test**: A visitor who exceeds the attempt budget receives a 429
response and can no longer submit further attempts within the window.

**Acceptance Scenarios**:

1. **Given** a visitor submits attempts to sign in, **When** the visitor exceeds
   the allowed attempts within the window, **Then** the system returns a 429
   response and rejects further attempts until the window passes.

## Edge Cases

- An email surrounded by whitespace is trimmed and then normalized before it is
  checked, so an email with surrounding spaces still matches its account.
- An email that is still malformed after trimming is rejected as invalid before any
  credential verification or token issuance happens.
- An unknown email and a wrong password for a known email produce the same
  failure outcome, so the response never reveals account existence.
- A rejected attempt never issues an access token or refresh token, and never
  returns credential data.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST normalize the email a learner submits by trimming
  surrounding whitespace and applying case normalization before validating its
  format and verifying the credentials, and MUST use that normalized email for both
  the lookup and the comparison.
- **FR-002**: The system MUST return a 200 response carrying an access token and a
  refresh token when the credentials are valid. The access token MUST be valid for
  `15m` and MUST identify the user and carry the user role. The refresh token MUST
  be valid for `7d`, MUST be persisted with a unique identifier, and MUST be issued
  as a token that rotates.
- **FR-003**: The system MUST return a 401 response with the generic code
  `INVALID_CREDENTIALS` when the email or password is invalid, and MUST NOT disclose
  whether an account exists for the email.
- **FR-004**: The system MUST allow at most 10 sign-in attempts per minute per
  visitor and MUST return a 429 response when the limit is exceeded.

## Key Entities

- **User**: A registered person. Carries the unique user identifier, the normalized
  email, the protected password representation, and the role the tokens carry.
- **Access token**: A short-lived credential issued at sign-in. Carries the user id
  and the role, and expires after `15m`.
- **Refresh token**: A longer-lived credential issued at sign-in. Carries a unique
  identifier and expires after `7d`; it is persisted and rotated on use.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A learner with valid credentials receives a 200 response with an
  access token and a refresh token within a single sign-in attempt.
- **SC-002**: Every failed sign-in returns a 401 with the generic code
  `INVALID_CREDENTIALS`, and no response reveals whether an email is registered.
- **SC-003**: A visitor who exceeds 10 attempts in a minute receives a 429 response,
  and further attempts within the window are rejected.
- **SC-004**: Every issued refresh token is persisted with a unique identifier and a
  `7d` expiry, and a token that has been rotated cannot be used again.

## Assumptions

- Sign-in verifies against the protected password recorded at registration, using
  the same password rules.
- The rate limit is keyed on the server-observed network address of the visitor.
- How the learner stores tokens between the access and refresh steps is out of scope
  for this feature.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The endpoint is `POST /api/identity/auth/login`.
- A successful sign-in returns `200`.
- The access token has a `15m` TTL.
- The refresh token has a `7d` TTL.
- An invalid email or password returns `401` with the code `INVALID_CREDENTIALS`.
- Exceeding the attempt limit returns `429`.
