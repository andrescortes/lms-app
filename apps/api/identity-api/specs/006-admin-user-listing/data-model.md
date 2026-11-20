# Data Model: Admin User Listing (CU-AUTH-06 / HU-17)

## Entities

### UserSummary (read model)

- Fields: `id`, `email`, `displayName`, `role`, `cefrLevel` (nullable), `createdAt`.
- Derived from the `users` table, selecting no password column.

### ListFilter (new)

- Fields: `Role` (optional role string), `CEFRLevel` (optional level string),
  `Search` (optional fragment), `Limit`, `Offset`.
- Validation: `Limit` between 1 and 100 inclusive; `Offset` non-negative; a role
  present must parse to a known role.

### UserPage (new, result)

- Fields: `Items []UserSummary`, `Total int`.
- `Total` is the count of accounts matching the filters, not the page length.

## State Transitions

The list reads only; no row changes.

| Input | Behaviour |
| --- | --- |
| no filters, page 1, limit 20 | first 20 accounts newest first, total of all |
| page past the end | empty items, unchanged total |
| search only spaces | treated as no search |
| unknown role | refused as a validation outcome |
| limit 0 or 101 | refused as a validation outcome |

Ordering is `created_at DESC, id ASC`, so the sequence is total and paging is stable.
