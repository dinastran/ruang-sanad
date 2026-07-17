---
type: entity
status: stub
---

# Goose (Database Migrations)

**Goose** is the database migration tool for Laju Go. Migrations run **automatically on startup**.

## Important Distinction

Use `go run github.com/pressly/goose/v3/cmd/goose@latest` (not the `goose` binary) to avoid confusion with the [Goose AI agent](https://block.github.io/goose/).

## Conventions

- One table per migration file
- Each file must have `-- +goose Up` and `-- +goose Down` sections
- Schema source directory: `migrations/`

## Commands

- `npm run db:migrate` / `make migrate` — run migrations
- `npm run db:refresh` / `make db-refresh` — reset database

## Source

Captured from [[sources/SRC-2026-07-06-001]].
