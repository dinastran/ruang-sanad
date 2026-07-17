---
type: concept
status: stub
---

# Single-Table Migration Convention

Every migration file in Laju Go must contain **exactly one table**.

## Rationale

1. **Isolation** — If a session table migration fails, the users table is still migrated
2. **Granular rollback** — `goose down` can rollback a specific table
3. **Clear history** — Each table has its own migration timestamp
4. **Debug-friendly** — sqlc reads `migrations/` for schema; separate files are easier to debug

## Required Structure

Every migration file must have `-- +goose Up` and `-- +goose Down` sections.

## Example

```
migrations/
├── 0001_create_users_table.sql
├── 0002_create_sessions_table.sql
└── 0003_create_password_resets_table.sql
```

## Source

Captured from [[sources/SRC-2026-07-06-001]].
