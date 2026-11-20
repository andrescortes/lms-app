# Feature Specification: Admin User Listing (CU-AUTH-06 / HU-17)

**Source**: CU-AUTH-06

**Created**: 2026-11-20

**Status**: Draft

## User Scenarios & Testing

### User Story 1 - Browse All Learners (Priority: P1)

An administrator opens the platform's user list to find an account. The list shows
learners newest first, in pages, with the total count so the administrator knows
how many accounts exist.

**Why this priority**: an administrator with no list has no way to find an account
at all. It is the whole point of the slice.

**Independent Test**: An administrator asks for the user list and receives the first
page of accounts newest first, plus the total number of accounts.

**Acceptance Scenarios**:

1. **Given** an administrator signed in, **When** they ask for the user list,
   **Then** the platform returns the first page of 20 accounts newest first with
   each account's identifier, email, display name, role, learning level, and
   creation date, and the total number of accounts.
2. **Given** fewer accounts exist than the page size, **When** the administrator
   asks for the list, **Then** every account is returned and the total matches.

### User Story 2 - Narrow the List to What I Need (Priority: P1)

An administrator is looking for teachers, or for a specific learner whose email
they half-remember. They filter the list by role, by learning level, or by a piece
of text that appears in an email or display name.

**Why this priority**: an unfiltered list is unusable once the platform has more
than a few dozen accounts.

**Independent Test**: With a mixed set of accounts, filtering by role returns only
that role, filtering by text returns only accounts whose name or email contains it
regardless of letter case, and filtering by learning level returns only that level.

**Acceptance Scenarios**:

1. **Given** accounts of several roles, **When** the administrator filters by one
   role, **Then** only accounts with that role are returned and the total reflects
   the filtered set.
2. **Given** accounts with different emails and display names, **When** the
   administrator searches for a fragment in any letter case, **Then** every account
   whose email or display name contains that fragment is returned.
3. **Given** accounts with different learning levels, **When** the administrator
   filters by one level, **Then** only accounts at that level are returned.
4. **Given** a filter that matches no account, **When** the administrator applies
   it, **Then** an empty list and a total of zero are returned.

### User Story 3 - Move Through Pages (Priority: P2)

An administrator pages through the list, asking for a specific page and page size.

**Why this priority**: paging only matters once the list is long, so it follows the
first two.

**Independent Test**: With more accounts than one page holds, asking for page two
returns the next accounts, no account appears on two pages, and the last page holds
the remainder.

**Acceptance Scenarios**:

1. **Given** more accounts than a page holds, **When** the administrator asks for
   page two with a fixed page size, **Then** the accounts returned start where page
   one ended.
2. **Given** a page past the end of the list, **When** the administrator asks for
   it, **Then** an empty list and the unchanged total are returned.

### User Story 4 - Only Administrators Can See the User List (Priority: P1)

A learner, or someone without a session, tries to reach the user list. The platform
refuses before any account data is read.

**Why this priority**: the list exposes every learner's email address. It must never
be reachable by a non-administrator.

**Independent Test**: A request without a session returns `401`; a signed-in learner
returns `403`; neither returns any account.

**Acceptance Scenarios**:

1. **Given** a request with no valid bearer credential, **When** it reaches the user
   list, **Then** the platform returns `401` and no accounts.
2. **Given** a signed-in learner who is not an administrator, **When** they request
   the user list, **Then** the platform returns `403` and no accounts.

### User Story 5 - Bad Filters Are Refused, Not Ignored (Priority: P2)

An administrator passes a page number of zero, a page size beyond the allowed
maximum, or a role that does not exist. The platform refuses and says which
parameter is wrong, rather than silently returning something else.

**Why this priority**: silently substituting defaults hides client bugs and makes a
mis-filtered list look complete.

**Independent Test**: Each invalid parameter is rejected with a validation outcome
naming it, and no accounts are returned.

**Acceptance Scenarios**:

1. **Given** a page of zero or a limit outside one to one hundred, **When** the
   administrator requests the list, **Then** the platform refuses with a validation
   outcome naming the parameter.
2. **Given** a role or learning level that does not exist, **When** the
   administrator requests the list, **Then** the platform refuses with a validation
   outcome naming the parameter.

## Edge Cases

- A search fragment of one character matches every account containing it; there is
  no minimum length.
- A search fragment that is only spaces is treated as no search, not as a search
  for spaces.
- Two accounts created in the same instant still appear in a stable order, so a
  page boundary cannot repeat or skip one.
- An account whose learning level is unset appears in no level-filtered result but
  does appear in an unfiltered list.
- A limit of exactly 1 and exactly 100 are both accepted; 0 and 101 are refused.

## Requirements

### Functional Requirements

- **FR-001**: The platform MUST return the user list to an authenticated
  administrator as a page of accounts, each carrying identifier, email, display
  name, role, learning level, and creation date, and MUST never include a
  password in any form.
- **FR-002**: The platform MUST order the list newest first and MUST break ties
  deterministically so that paging never repeats or skips an account.
- **FR-003**: The platform MUST report the total number of accounts matching the
  filters, independent of which page was requested.
- **FR-004**: The platform MUST support filtering by role, by learning level, and by
  a text fragment matched case-insensitively against email or display name, and MUST
  apply those filters to both the page and the total.
- **FR-005**: The platform MUST default to page one with a page size of 20, MUST
  accept a page size from 1 to 100 inclusive, and MUST refuse any other value with a
  validation outcome naming the parameter.
- **FR-006**: The platform MUST refuse a role or learning level that does not exist
  with a validation outcome naming the parameter, rather than ignoring it.
- **FR-007**: The platform MUST require an authenticated caller whose role is
  administrator, returning `401` without a valid credential and `403` for a
  signed-in caller who is not an administrator, before any account is read.

## Key Entities

- **User summary**: One account as it appears in the list. Carries identifier,
  email, display name, role, learning level, and creation date, and no password.
- **User page**: One page of user summaries plus the total count of accounts matching
  the filters, the page number returned, and the page size used.

## Success Criteria

### Measurable Outcomes

- **SC-001**: An administrator receives the first page of the list with a total count
  in a single request.
- **SC-002**: 100% of list responses contain no password material in any form, on
  success or on error.
- **SC-003**: Paging through the whole list in sequence returns every account exactly
  once, in newest-first order.
- **SC-004**: Every filter narrows both the returned page and the reported total
  consistently; 100% of unfiltered accounts appear in the unfiltered list.
- **SC-005**: Every invalid filter or pagination parameter is refused with a
  validation outcome naming it; 100% of non-administrators receive `403`.

## Assumptions

- The administrator is any signed-in caller whose role is administrator; there is no
  separate admin permission in this slice.
- The list is read-only; editing or deleting accounts is a later concern.
- Learning-level values are free text at registration, so filtering by level matches
  the stored value exactly rather than validating against a fixed scale.

## Verbatim Constraints

The following literals are fixed by the request and MUST be matched exactly:

- The endpoint is `GET /api/identity/users`.
- The success body is `{"items": [{"id", "email", "displayName", "role", "cefrLevel", "createdAt"}], "total": N, "page": P, "limit": L}`.
- The default page is `1`; the default limit is `20`; the limit range is `1` to `100`.
- Query parameters are `role`, `cefrLevel`, `q`, `page`, `limit`.
- A caller who is not an administrator receives `403` with `FORBIDDEN`.
- An unauthenticated caller receives `401`.
- Invalid filters or pagination return `400` or `422`.
