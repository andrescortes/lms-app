# Edit Course — Research

## Decisions

| Decision | Rationale | Alternatives considered |
|---|---|---|
| Domain-first data model (`internal/domain` only) | Constitution II requires `internal/domain` to import only the Go standard library; moving validation into an adapter would violate that constraint. | Validation in adapter — rejected for violating Constitution II. |
| SQL queries via `sqlc generate` (engine `sqlite`) | Constitution III mandates `sqlc generate` with `engine: "sqlite"`, `db/queries/` → `internal/adapter/sqlite/sqlcgen/`. | Hand-written `database/sql` queries — rejected for violating Constitution III. |
| Local JWT verification (`HS256` via `JWT_SECRET`, no identity-service) | Constitution IV requires JWT verification locally, extracting `sub`/`role`, without calling `identity-service` on the hot path. | Per-request identity-service HTTP call — rejected for violating Constitution IV. |
| Composable chi middlewares (`Authenticate`, `RequireRole`, `RequireCourseOwner`) | Constitution II names these middlewares and requires `net/http` handlers. | Framework-level auth wrappers — rejected for bypassing the chi middleware requirement. |

## Unknowns / NEEDS CLARIFICATION

None. All design decisions are resolved: the request is explicit about every constraint (domain, DB, auth, contracts, error handling), and the project constitution provides unambiguous governance for each.

## Clarification markers

Zero `[NEEDS CLARIFICATION: ...]` markers remain after the plan's constitution check. All requirements map directly to implementation choices already recorded.