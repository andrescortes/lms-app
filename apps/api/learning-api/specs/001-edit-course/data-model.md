# Edit Course — Data Model

## Entities

### Course

| Field | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `string` (UUID) | PRIMARY KEY, NOT NULL | Generated on create; immutable |
| `teacher_id` | `string` (UUID) | NOT NULL, FK → users(id) | Owner; determines edit permission |
| `title` | `string` | NOT NULL, non-empty after trim | Required; validation: trimmed length > 0 |
| `description` | `string` | NULL allowed, default `''` | Optional free-text |
| `cefr_level` | `string` | NOT NULL, CHECK IN (`'A1','A2','B1','B2','C1','C2'`) | ISO-like CEFR enum; case-sensitive |
| `cover_url` | `string` | NULL allowed, default `''` | Optional HTTPS URL |
| `status` | `string` | NOT NULL, default `'draft'` | Not editable via this feature |
| `created_at` | `string` (ISO-8601 UTC) | NOT NULL, TEXT, UTC | Set on create; immutable |

## Relationships

- `Course.teacher_id` → `User.id` (foreign key). The `users` table is managed by `identity-service` but the FK exists locally for ownership checks. `learning-api` does not mutate `users`.

## Validation Rules (from `internal/domain/course.go`)

- `Title`: MUST NOT be empty after `strings.TrimSpace`. Violation → `domain.ErrValidation`.
- `CEFRLevel`: MUST be one of `A1`, `A2`, `B1`, `B2`, `C1`, `C2` (case-sensitive). Violation → `domain.ErrValidation`.
- `Description`, `CoverURL`: free text/URL; no format validation at domain layer (schema validation happens at the edge).

## Sentinel Errors (from `internal/domain/errors.go`)

| Error | Meaning | HTTP mapping via `respondError` |
|---|---|---|
| `ErrValidation` | Domain validation failed | `422` |
| `ErrNotFound` | Course not found | `404 NOT_FOUND` |
| `ErrForbidden` | Caller lacks permission (wrong role or non-owner) | `403 FORBIDDEN` |
| `ErrInvalidToken` | JWT malformed/expired/unsigned | `401` (via `Authenticate` middleware) |
| `ErrConflict` | Optimistic-lock or unique violation (not used by this feature) | `409` |

## State Transitions (for this feature)

None. This feature only mutates metadata fields (`title`, `description`, `cefr_level`, `cover_url`). The `status` field is not edited here.

## Schema DDL (from `db/migrations/001_courses.up.sql`)

```sql
CREATE TABLE courses (
  id TEXT PRIMARY KEY NOT NULL,
  teacher_id TEXT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL CHECK (trim(title) != ''),
  description TEXT NOT NULL DEFAULT '',
  cefr_level TEXT NOT NULL CHECK (cefr_level IN ('A1','A2','B1','B2','C1','C2')),
  cover_url TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft',
  created_at TEXT NOT NULL
) STRICT;
-- Index for owner lookups
CREATE INDEX idx_courses_teacher_id ON courses(teacher_id);
```

Down migration (`001_courses.down.sql`): `DROP TABLE IF EXISTS courses;`

All timestamps stored as ISO-8601 UTC `TEXT` per Constitution III.