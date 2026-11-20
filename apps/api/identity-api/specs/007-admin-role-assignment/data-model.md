# Data Model: Admin User Role Assignment (CU-AUTH-07 / HU-17)

## Entities

### Account (existing `users` table)

- Field changed: `role`.
- Fields returned: `id`, `email`, `display_name`, `role`, `cefr_level`, `created_at`.
- Never returned: `password_hash`.

### RoleAssignment (new, request-scoped)

- Fields: `ActorUserID` (from the verified token), `TargetUserID` (from the path),
  `NewRole` (from the body).
- Validation: `NewRole` must be one of STUDENT, TEACHER, ADMIN, REVIEWER exactly;
  `ActorUserID` must differ from `TargetUserID`.

## State Transitions

| Request | Account role |
| --- | --- |
| valid role, actor differs from target | set to the new role |
| valid role equal to the current role | unchanged, still reported as success |
| unknown role | unchanged, refused |
| actor is the target | unchanged, refused |
| target does not exist | nothing to change, refused |

Every refusal is decided before the write, so a refused request never partially
applies.
