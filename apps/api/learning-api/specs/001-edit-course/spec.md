# Edit Course

**Source**: CU-CONT-02 / HU-08 CA5

## User Scenarios & Testing

### User Story 1 - Teacher edits their own course metadata (Priority: P1)

A teacher opens an existing course they own and changes its title, description, CEFR level,
or cover image. Only the fields they send change; everything else stays as it was.

**Why this priority**: This is the core value of the feature. Without it there is nothing to
protect or validate.

**Independent Test**: Send a request that changes one field for a course the teacher owns and
confirm the response returns the updated course while untouched fields keep their old values.

**Acceptance Scenarios**:

1. **Given** a teacher owns a course, **When** they send a partial update changing only the
   title, **Then** the request succeeds and the response carries the new title while
   description, CEFR level, and cover image are unchanged.
2. **Given** a teacher owns a course, **When** they send updates for title, description, CEFR
   level, and cover image together, **Then** all four fields are updated and returned.
3. **Given** a course update succeeds, **When** the teacher reloads the course, **Then** the
   persisted values match the update response.

### User Story 2 - Only the owning teacher may edit a course (Priority: P2)

A teacher cannot edit a course owned by someone else, and users without the teacher role cannot
edit courses at all.

**Why this priority**: Ownership protects course content from unauthorized edits.

**Independent Test**: Attempt an update as a non-owner teacher and as a non-teacher and confirm
each is rejected while the owner still succeeds.

**Acceptance Scenarios**:

1. **Given** a course owned by another teacher, **When** a teacher sends an update, **Then** the
   request is forbidden and the course is unchanged.
2. **Given** a signed-in user without the teacher role, **When** they send an update, **Then** the
   request is forbidden.
3. **Given** a request with no valid credentials, **When** it targets any course, **Then** it is
   rejected as unauthenticated.

### User Story 3 - Invalid updates are rejected without changing data (Priority: P3)

Malformed or invalid updates never overwrite a course. The system reports what was wrong.

**Why this priority**: Guards stored data against bad input, but depends on the update path
existing first.

**Independent Test**: Send each invalid payload and confirm a validation error is returned and
the stored course is unchanged.

**Acceptance Scenarios**:

1. **Given** an update with an empty or whitespace-only title, **When** it is sent, **Then** it is
   rejected as a validation error.
2. **Given** an update with a CEFR level outside the supported set, **When** it is sent, **Then**
   it is rejected as a validation error.
3. **Given** a request body that is not valid JSON, **When** it is sent, **Then** it is rejected as
   a bad request.
4. **Given** an update targets a course that does not exist, **When** it is sent, **Then** it is
   reported as not found.

## Edge Cases

- Only some fields are provided: only those fields change.
- Title present but empty or whitespace: rejected.
- CEFR level casing or value not in the supported set: rejected.
- Course id does not exist: not found.
- Caller is a teacher but not the owner: forbidden.
- Caller is authenticated but not a teacher: forbidden.
- Body is malformed JSON: bad request.
- All four optional fields omitted: the course is returned unchanged.

## Requirements

### Functional Requirements

- **FR-001**: The system MUST expose course metadata updates at `PATCH /api/learning/courses/{id}`.
- **FR-002**: The system MUST update only the fields present in the request and leave the others
  unchanged.
- **FR-003**: The system MUST reject a title that is empty or whitespace-only with a validation
  error.
- **FR-004**: The system MUST reject a CEFR level outside `A1`, `A2`, `B1`, `B2`, `C1`, `C2` with
  a validation error.
- **FR-005**: The system MUST return `404 NOT_FOUND` when the target course does not exist.
- **FR-006**: The system MUST return `403 FORBIDDEN` when the caller is not the course owner.
- **FR-007**: The system MUST require the `TEACHER` role and return `403 FORBIDDEN` for other
  authenticated roles.
- **FR-008**: The system MUST require authentication and reject missing or invalid credentials.
- **FR-009**: The system MUST return `400` on a malformed JSON body.
- **FR-010**: On success the system MUST return `200 OK` with the fields `id`, `teacherId`,
  `title`, `description`, `cefrLevel`, `coverUrl`, `status`, and `createdAt`.
- **FR-011**: Errors MUST be returned through the shared JSON error envelope with `code`,
  `message`, `timestamp`, and `path`.

### Key Entities

- **Course**: a teaching unit with `id`, `teacherId`, `title`, `description`, `cefrLevel`,
  `coverUrl`, `status`, and `createdAt`. The title is required and non-empty; the CEFR level is
  one of the supported values; teacher ownership determines who may edit it.

## Success Criteria

### Measurable Outcomes

- **SC-001**: An owner can update any subset of the four metadata fields and receive the updated
  course in one request.
- **SC-002**: 100% of updates from non-owners and non-teachers are rejected without changing data.
- **SC-003**: 100% of invalid titles, invalid CEFR levels, malformed bodies, and missing courses
  return the documented error code and leave the stored course unchanged.
- **SC-004**: A reload after a successful update shows the same values that were returned.

## Assumptions

- Partial updates follow merge semantics: absent fields are left unchanged.
- Title validation applies to the resulting title, trimmed.
- Courses already carry a `status`; editing metadata does not change it.

## Verbatim Constraints

- Endpoint: `PATCH /api/learning/courses/{id}`
- Required role: `TEACHER`
- CEFR levels: `A1`, `A2`, `B1`, `B2`, `C1`, `C2`
- Error codes: `NOT_FOUND`, `FORBIDDEN`
- Response fields: `id`, `teacherId`, `title`, `description`, `cefrLevel`, `coverUrl`, `status`,
  `createdAt`
- JWT secret env var: `JWT_SECRET`
