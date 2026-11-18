# Feature Specification: Student Registration (CU-AUTH-01 / HU-01)

**Feature Branch**: `[001-student-registration]`

**Created**: 2026-11-18

**Status**: Draft

**Input**: User description: "As a visitor, I want to register an account with an email and password so that I can access the English learning platform. CA1: validate email format and uniqueness (case-insensitive normalization). CA2: password at least 8 characters, 1 uppercase letter, 1 numeric digit; never stored or returned in plain text. CA3: create User with default role STUDENT and empty Profile record (native language, learning goals, timezone, avatar URL unset). CA4: 201 with userId on success; 409 structured conflict without sensitive details; 400/422 validation errors detailing the violated rule. CA5: emit internal UserRegistered event (userId, email, role)."

## Clarifications

### Session 2026-11-18

- Q: If the internal user-registered notification cannot be delivered after a
  successful registration, should the registration itself be failed and rolled
  back, or should it succeed with the notification retried? → A: Registration
  succeeds regardless; the notification is delivered best-effort with
  automatic retries until it lands.
- Q: Should the system limit how many registration attempts a single visitor
  can make in a short period to slow down automated abuse of the signup flow? →
  A: Per-visitor limit: a visitor who submits too many registrations in a short
  window is temporarily locked out of trying again.
- Q (follow-up on FR-011): How many registration attempts per short window
  trigger the lockout, and how long does the lockout last? → A: 5 attempts per
  15 minutes triggers a 30-minute lockout.
- Q: When a locked-out visitor submits another registration attempt, should
  the response reveal how much lockout time remains, or only that they are
  rate-limited? → A: The response only says the visitor is rate-limited,
  without revealing remaining lockout time.
- Q: Must the promise that every successful registration's `UserRegistered`
  notification eventually lands still hold after a service restart? → A: Yes —
  the event is recorded and delivery resumes after a restart; "exactly one"
  means one logical event per registration, and consumers tolerate rare
  duplicate deliveries.
- Q: When a visitor reaches the registration attempt limit, is it the fifth
  attempt within 15 minutes that triggers the 30-minute lockout, or only an
  attempt beyond the fifth? → A: The first five attempts within 15 minutes are
  allowed; the sixth attempt triggers the 30-minute lockout.
- Q: When a visitor enters a well-formed email surrounded by leading or
  trailing spaces, should the system accept it after removing those spaces, or
  reject it as invalid? → A: Accept it after trimming the surrounding spaces,
  then normalize case and store the trimmed form.
- Q: Is there a maximum accepted password length, or are passwords accepted at
  any length as long as they meet the 8-character, uppercase, and digit rules? →
  A: Yes — accept 8 through 72 characters; a longer password is rejected with a
  validation error naming the length rule.
- Q: Should SC-004's requirement that every rejection name the specific
  violated rule apply to all rejection outcomes, or only to validation and
  duplicate-conflict rejections? → A: Only validation failures and
  duplicate-email conflicts must name the specific rule; other outcomes use
  their own standard codes and messages.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register a New Student Account (Priority: P1)

A new visitor to the English learning platform creates an account by providing an
email address and a password that meets the password policy. The system checks
that the email is well-formed and not already registered, then creates a student
account and an empty learner profile so the visitor can sign in and start
learning.

**Why this priority**: Registration is the platform's front door. No visitor
can access any learning functionality before completing it, so it is the
minimum viable slice of the feature.

**Independent Test**: A visitor submits a valid, unused email and a compliant
password and receives a success confirmation containing their new user
identifier; their account and empty profile are available immediately for
sign-in.

**Acceptance Scenarios**:

1. **Given** a visitor who has no account and the email is not registered,
   **When** the visitor submits a well-formed email and a password of at least
   8 characters containing at least one uppercase letter and one numeric
   digit, **Then** the system confirms account creation, returns the new user
   identifier, and assigns the account the default student role with an
   empty profile (no native language, learning goals, timezone, or avatar
   URL set).
2. **Given** a visitor who has no account, **When** the visitor submits an
   email that differs from an existing registered email only in letter
   case, **Then** the system reports that the email is already registered and
   does not create a duplicate account.
3. **Given** a visitor completing registration, **When** the account is
   successfully created, **Then** an internal user-registered notification is
   emitted carrying the new user's identifier, email, and role, so that
   downstream learning workflows can react to the new student.

---

### User Story 2 - Register with an Invalid Email (Priority: P2)

A visitor submits an email that is missing a domain, contains invalid
characters, or is otherwise not a well-formed address. The system rejects the
attempt, points to the specific violated email rule, and creates no account.

**Why this priority**: Protecting account data quality is essential, but it
is a rejection path of the P1 flow rather than an independent journey, so it
is a second-priority slice.

**Independent Test**: A visitor submits malformed emails (no `@`, no domain,
trailing dot, empty local part) and observes the system name the violated
email rule without creating any account or emitting any notification.

**Acceptance Scenarios**:

1. **Given** a visitor with no account, **When** the visitor submits an email
   that is not a well-formed address, **Then** the system rejects the
   registration with a validation outcome that identifies the email format
   rule that was violated.

---

### User Story 3 - Register with a Weak Password (Priority: P2)

A visitor submits a password that is too short or lacks an uppercase letter or
a numeric digit. The system rejects the attempt and names the violated policy
rule, so the visitor can correct the password and retry. No account is created.

**Why this priority**: Password-policy enforcement is part of the P1 journey,
but the rejection path is independently testable and worth its own priority
because it guards the account-strength invariant for all future sign-ins.

**Independent Test**: A visitor submits passwords of 7 characters, all
lowercase with digits, and digits without letters, and observes each rejection
name the specific violated policy rule.

**Acceptance Scenarios**:

1. **Given** a visitor with no account, **When** the visitor submits a
   password shorter than 8 characters, or one with no uppercase letter, or
   one with no numeric digit, **Then** the system rejects the registration
   with a validation outcome that identifies the violated password rule, and
   the submitted password is never kept or shown back to the visitor.

---

### User Story 4 - Register with an Already-Registered Email (Priority: P3)

A visitor submits a valid email that another account already uses, in any
letter case. The system rejects the attempt with a conflict outcome that uses
a standard error code and does not reveal details of the existing account.

**Why this priority**: Conflict handling prevents silent duplicates, but it
only matters once accounts exist, so it lands after the creation and
validation paths.

**Independent Test**: A visitor submits an email that exactly matches, or
differs only in case from, an existing registered email and observes a
conflict response with a standard error code and no leakage of the existing
account's details.

**Acceptance Scenarios**:

1. **Given** an existing registered email, **When** a visitor submits that
   email in any letter case during registration, **Then** the system returns a
   conflict outcome with a standard error code and does not disclose details
   of the existing account.

---

### Edge Cases

- **Case-insensitive email handling**: `Student@Example.com` and
  `student@example.com` are the same email; uniqueness is checked and the
  email is kept using one consistent normalized form.
- **Whitespace or unusual characters**: leading and trailing spaces around the
  email are trimmed before validation, case normalization, uniqueness checking,
  and storage. A well-formed email with surrounding spaces is accepted once
  trimmed; an address that is still malformed after trimming fails validation.
- **Password at exactly 8 characters**: a password of exactly 8 characters
  that contains at least one uppercase letter and one digit is accepted.
- **Special characters in password**: a password meeting the policy is
  accepted even if it contains symbols, spaces, or accented characters; the
  policy checks only length, uppercase, and digit requirements.
- **Password above the maximum**: a password longer than 72 characters is
  rejected with a validation error naming the length rule; a password of
  exactly 72 characters meeting the other rules is accepted.
- **Simultaneous registration of the same email**: if two visitors submit the
  same email at the same moment, exactly one account is created and the other
  visitor receives a conflict outcome.
- **Concurrent duplicate email attempts**: duplicate email submissions never
  leave behind partial or half-created accounts; a rejected attempt creates
  nothing and emits no notification.
- **Failed user-registered notification delivery**: a new account whose
  user-registered notification cannot be delivered is not rolled back; the
  visitor's registration succeeds, and the notification is recorded and
  retried automatically (including across a service restart) until it is
  delivered, so no successful registration loses its notification permanently.
  Duplicate deliveries of the same logical event are tolerated by consumers.
- **Rejected attempt does not leak account existence beyond the conflict
  code**: conflict responses reveal only that an account exists for that
  email, not the stored email, creation date, or any profile data.
- **Rate-limited visitor**: the first five registration attempts within 15
  minutes are allowed; a visitor who makes a sixth attempt in that window is
  temporarily locked out for 30 minutes and cannot submit further attempts
  until the lockout expires; the lockout applies per-visitor, not globally.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST accept a registration request consisting of an
  email address and a password, and MUST normalize the email by trimming
  surrounding whitespace and applying case-insensitive normalization before
  validating its format, checking uniqueness against existing accounts, and
  storing it. The normalized form MUST be what is stored and compared.
- **FR-002**: The system MUST reject a registration whose email is not a
  well-formed address, and MUST name the email-format rule that was
  violated in the validation outcome.
- **FR-003**: The system MUST accept a password of 8 through 72 characters
  (inclusive) that contains at least one uppercase letter and at least one
  numeric digit. It MUST reject a password shorter than 8 characters, longer
  than 72 characters, containing no uppercase letter, or containing no numeric
  digit, and MUST name the violated password rule in the validation outcome.
- **FR-004**: The system MUST reject a registration whose normalized email is
  already registered, with a conflict outcome that carries a standard error
  code and does not disclose details of the existing account.
- **FR-005**: The system MUST keep the submitted password only in a
  protected, non-reversible form, and MUST never store it, send it back to
  the visitor, or include it in any account details, notification, or error
  message.
- **FR-006**: On a successful registration, the system MUST create a user
  account that carries a unique identifier, the normalized email, and the
  default student role, and MUST assign the account an empty profile with
  native language, learning goals, timezone, and avatar URL all unset.
- **FR-007**: The successful registration outcome MUST include the new
  user's identifier so the visitor can proceed to sign in.
- **FR-008**: The system MUST reject two simultaneous registrations of the
  same normalized email so that at most one account exists for any email.
- **FR-009**: On every successful registration, the system MUST emit exactly
  one logical internal user-registered notification that carries the new
  user's identifier, the normalized email, and the assigned role. A
  notification delivery failure MUST NOT block or roll back the registration;
  the notification MUST be recorded so delivery resumes after a service
  restart and MUST be retried automatically until delivered. "Exactly one"
  means one logical event per registration; downstream consumers MUST tolerate
  rare duplicate deliveries of the same logical event.
- **FR-010**: The system MUST NOT emit a user-registered notification, nor
  create any account, for a rejected registration attempt.
- **FR-011**: The system MUST apply a per-visitor limit on registration
  attempts: the first five registration attempts within any 15-minute window
  MUST be allowed, and from the sixth attempt onward the visitor MUST be
  temporarily locked out for 30 minutes from submitting further attempts. The
  lockout MUST apply per visitor and MUST NOT slow down or lock out other
  visitors. A submission while locked out MUST be rejected with an outcome that
  states the visitor is rate-limited, without revealing how much lockout time
  remains.

### Key Entities *(include if feature involves data)*

- **User**: A registered person on the platform. Carries a unique user
  identifier, the normalized email, the protected (non-reversible) password
  representation, and an assigned role. One user has exactly one profile.
- **Profile**: A learner's personal record linked to exactly one user. Holds
  native language, learning goals, timezone, and avatar URL, all of which
  start unset at registration and are filled in later.
- **Role**: The capability tier assigned to a user. Registration always
  assigns the default student role.
- **UserRegistered notification**: An internal signal emitted when a user is
  created. Carries the user identifier, the normalized email, and the role.
- **Registration attempt**: A single request to create an account, carrying
  an email and a password.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A visitor with a valid, unused email and a compliant password
  completes registration in one submission and receives the user identifier
  in under 3 seconds under normal conditions.
- **SC-002**: 100% of registered accounts have the student role and an
  empty profile (no native language, learning goals, timezone, or avatar
  URL set).
- **SC-003**: No email, in any letter case, appears more than once among
  registered accounts; every conflicting duplicate submission is rejected
  with the conflict outcome.
- **SC-004**: 100% of validation failures and duplicate-email conflicts are
  met with an outcome that names the specific violated rule (email format,
  password length, uppercase, digit, or duplicate email) and reveals no
  existing account's details. Rate-limited, malformed-request, and unexpected
  rejection outcomes use their own standard codes and messages instead.
- **SC-005**: 100% of successful registrations produce exactly one logical
  user-registered notification carrying the identifier, normalized email, and
  role; a restart before delivery does not lose it, and duplicate deliveries
  of the same logical event do not cause duplicate downstream effects.
- **SC-006**: A reviewer auditing stored account data finds no plaintext
  password for any registered user, across all outcomes (success,
  validation failure, or conflict).
- **SC-007**: A visitor who exceeds the per-visitor registration-attempt
  limit receives a temporary lockout outcome, and no other visitor's
  registration attempts are affected by that lockout.

## Assumptions

- Visitors register through the platform's own account flow; there is no
  email verification or confirmation link in this feature.
- Registration assigns the student role only; account creation never
  assigns any other role.
- Profile fields (native language, learning goals, timezone, avatar URL)
  are filled in by a later feature; they stay unset at registration.
- A single logical user-registered notification is emitted once per successful
  registration, not once per rejected attempt; it is recorded and retried
  until delivered, including after a service restart, so a temporary
  notification failure does not prevent the registration from succeeding.
  Downstream consumers tolerate rare duplicate deliveries.
- The visitor receives only the user identifier on success; no credentials,
  email address, or profile data are returned.
- Simultaneous duplicate submissions are a rare but real case; the system
  must never create two accounts for the same email.
