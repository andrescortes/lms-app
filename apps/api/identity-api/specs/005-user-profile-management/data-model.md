# Data Model: User Profile Management (CU-AUTH-05 / HU-18)

## Entities

### Account (existing `users` table)

- Fields: `id`, `email`, `password_hash`, `display_name`, `role`, `cefr_level`,
  `created_at`.
- Editable through this feature: `display_name` and `password_hash`.
- Never exposed: `password_hash`.

### Profile (existing `profiles` table)

- Fields: `user_id` (primary key, references `users`), `native_language`, `goals`,
  `timezone`, `avatar_url`, all nullable.
- Every field is optional to the learner; an unset field is reported empty.

### ProfileView (new, read model)

- The account fields plus the profile fields, assembled for the response.
- Carries no password material.

## Validation Rules

- A display name, when supplied, must be non-blank after trimming.
- A new password must satisfy the existing policy: 8 to 72 characters with at least
  one uppercase letter and one digit.
- A password change requires a current password that verifies against the stored
  hash.

## State Transitions

| Request shape | Account | Profile |
| --- | --- | --- |
| field absent | unchanged | unchanged |
| field `null` | not applicable (account fields are not nullable) | cleared |
| field with a value | set to the trimmed value | set to the value |
| new password with correct current | hash replaced | unchanged |
| new password with wrong or missing current | unchanged, refused `401` | unchanged |
| weak new password | unchanged, refused `422` | unchanged |

The account and profile writes commit together or not at all.
