---
type: entity
status: stub
---

# sqlc

**sqlc** generates type-safe Go code from SQL queries. In Laju Go, it produces the `app/queries/` package from `queries/*.sql`.

## Role

- Reads schema from `migrations/` directory (configured in `sqlc.yaml`)
- Generates typed Go query functions — **never edit `app/queries/` manually**
- Run via: `npm run db:generate` or `make db-generate`

## Source

Captured from [[sources/SRC-2026-07-06-001]].
