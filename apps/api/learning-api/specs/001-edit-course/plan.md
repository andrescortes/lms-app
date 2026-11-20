## Summary

This feature adds course metadata editing at `PATCH /api/learning/courses/{id}` for the `learning-api` Go service. A teacher who owns a course can update any subset of `title`, `description`, `cefrLevel`, and `coverUrl`; only the fields present change, invalid input is rejected without mutating data, and only the owning teacher may edit. The implementation follows the project constitution: `internal/domain` is stdlib-only, all HTTP handlers use chi/`net/http` with `log/slog` structured logging, persistence uses pure-Go `modernc.org/sqlite` with `goose` migrations and `sqlc`-generated queries, and JWTs are verified locally with HS256 via `JWT_SECRET`.

## Constitution Check

This gate is checked before Phase 0 research and re-checked after Phase 1 design.

| Constitution principle | Assessment | Evidence |
|---|---|---|
| I. Pure-Go Static Build | PASS | Build with `CGO_ENABLED=0`, Go 1.23+, `github.com/go-chi/chi/v5`, `net/http` handlers, `log/slog` with `request_id` |
| II. Clean Architecture One-Way Deps | PASS | `internal/domain` (stdlib only) → `internal/usecase` → `internal/adapter/*` → `cmd/api/main.go` wiring |
| III. SQLite Durability & Schema Discipline | PASS | `modernc.org/sqlite`, PRAGMAs WAL/NORMAL/foreign_keys=ON/busy_timeout=5000, `SetMaxOpenConns(1)`, goose migrations, STRICT tables, ISO-8601 UTC TEXT timestamps, INTEGER booleans, `CHECK json_valid(...)`, `sqlc generate` engine sqlite `db/queries/` → `internal/adapter/sqlite/sqlcgen/` |
| IV. Local Security & Contract Validation | PASS | `github.com/golang-jwt/jwt/v5` HS256 via `JWT_SECRET`, extracts `sub`/`role` locally (no identity-service hot path); JSON schemas via `github.com/santhosh-tekuri/jsonschema/v6` against `shared/contracts/exercise-schemas/` |
| V. Realtime Integrity & Bounded Lifecycles | PASS | Not directly used by this feature (SSE / WebSockets are for other features); no violation introduced |

No violations are necessary, so **Complexity Tracking** is omitted.

## Project Structure

```
internal/domain/
  course.go          Course struct + validation + sentinel errors (ErrValidation, ErrNotFound, ErrForbidden)
internal/usecase/
  update_course.go   UpdateCourseUseCase + UpdateCourseCommand
internal/adapter/sqlite/
  course_store.go    GetByID / Update, mapping sql.ErrNoRows -> domain.ErrNotFound
internal/adapter/http/
  course_handlers.go RequireCourseOwner middleware + PATCH /api/learning/courses/{id} handler
cmd/api/main.go      wires UpdateCourseUseCase, RequireCourseOwner, graceful shutdown
db/queries/courses.sql GetCourseByID + UpdateCourse sqlc queries
db/migrations/
  001_courses.up.sql / 001_courses.down.sql   goose migration (STRICT table)
shared/contracts/learning-api.yaml   OpenAPI contract updated
contracts/learning.openapi.yaml      OpenAPI contract updated
```

**Structure Decision**: The feature creates one new Go module surface under `internal/` plus migration/SQL/query artifacts under `db/`, contracts under `shared/contracts/` + `contracts/`, and handlers+wiring under `internal/adapter/http/` + `cmd/api/`. No existing files are edited except `db/queries/courses.sql` (append queries), `shared/contracts/learning-api.yaml`, `contracts/learning.openapi.yaml`, and `cmd/api/main.go` (wire the new use case + middleware).

## Phase 0: Research

Decisions recorded below. The codebase is greenfield (`learning-api` has only `go.mod`), so there are no existing patterns to copy; every file is new except the constitution-mandated build/test choices.

**Decision: Domain-first data model**
- Rationale: `internal/domain` must be stdlib-only per Constitution II; `Course` and its validation live there, and adapters depend on it.
- Alternatives considered: Putting validation in the adapter — rejected because Constitution II requires domain to hold pure domain logic including SM-2 SRS-style validation rules here generalized to metadata validation.

**Decision: SQL queries via sqlc**
- Rationale: Constitution III mandates `sqlc generate` with engine `sqlite`, `db/queries/` → `internal/adapter/sqlite/sqlcgen/`.
- Alternatives considered: hand-written `database/sql` — rejected because it violates Constitution III.

**Decision: JWT verification local, no identity-service call**
- Rationale: Constitution IV requires verifying JWTs locally with HS256 via `JWT_SECRET`, extracting `sub`/`role`, without calling identity-service on the hot path.
- Alternatives considered: HTTP call to identity-service per request — rejected because it violates Constitution IV.

**Decision: composable chi middlewares**
- Rationale: Constitution II names `Authenticate`, `RequireRole`, `RequireCourseOwner` as composable chi middlewares; Constitution uses `net/http` handlers only.
- Alternatives considered: framework-level auth wrappers — rejected because they would bypass the chi middleware requirement.

## Phase 1: Design and Contracts

### Data model

- `domain.Course`: fields `id`, `teacher_id`, `title`, `description`, `cefr_level`, `cover_url`, `status`, `created_at`.
- Validation rules: `Title` non-empty after trimming; `CEFRLevel` in `{A1,A2,B1,B2,C1,C2}`.
- Sentinel errors: `ErrValidation`, `ErrNotFound`, `ErrForbidden`, `ErrInvalidToken`, `ErrConflict`.

### Contracts (`contracts/`)

- `PATCH /api/learning/courses/{id}` under `Authenticate` + `RequireRole("TEACHER")` + `RequireCourseOwner`.
- Request body: JSON object with optional fields `title`, `description`, `cefrLevel`, `coverUrl`; all present → full update; subset → partial update; none → unchanged.
- Success `200 OK` response: `{"id", "teacherId", "title", "description", "cefrLevel", "coverUrl", "status", "createdAt"}`.
- Errors via `respondError`: `ErrNotFound` → `404 NOT_FOUND`, `ErrForbidden` → `403 FORBIDDEN`, `ErrValidation` → `422`, malformed JSON → `400`. Envelope fields: `code`, `message`, `timestamp`, `path`.

All identifiers copied verbatim from the spec's Verbatim Constraints.

## Phase 0 / Phase 1 constitution re-check

The design introduces no deviation from the constitution; all five principles are satisfied as tabled above. **Complexity Tracking** remains omitted.

## Capture (plan step)

```bash
python3 .specify/extensions/companion/scripts/write-context.py --feature-dir specs/001-edit-course --step plan --batch '{
  "decisions": [
    {"decision": "Domain-first data model: Course + validation live in stdlib-only internal/domain", "why": "Constitution II requires domain to hold pure domain logic; adapters depend on it", "rejected": "validation in adapter"},
    {"decision": "SQL queries via sqlc generate, engine sqlite", "why": "Constitution III mandates sqlc; db/queries/ -> internal/adapter/sqlite/sqlcgen/", "rejected": "hand-written database/sql"},
    {"decision": "Local JWT verification, HS256 via JWT_SECRET, no identity-service hot path", "why": "Constitution IV forbids identity-service calls on the hot path", "rejected": "per-request identity-service call"},
    {"decision": "Composable chi middlewares Authenticate / RequireRole / RequireCourseOwner", "why": "Constitution II names these middlewares and net/http handlers only", "rejected": "framework-level auth wrappers"}
  ],
  "step_summary": {"summary": "Plan for PATCH /api/learning/courses/{id}; domain-first model, sqlc, local JWT, chi middlewares", "key_finding": "learning-api is greenfield; every file is new except build/test tooling"},
  "set": {"approach": "Implement domain Course + validation, sqlc GetCourseByID/UpdateCourse, sqlite course store, usecase with ownership check, chi handler + RequireCourseOwner, OpenAPI update, wiring, tests."}
}'
```

**The next step is `tasks`.** Dispatch `speckit.companion.tasks <feature_dir>`.
