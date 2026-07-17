---
type: concept
status: stub
---

# Code Generation Pipeline

Laju Go uses two code generation tools to produce type-safe Go code:

## sqlc

- Input: `queries/*.sql` + schema from `migrations/`
- Output: `app/queries/` (typed Go query functions)
- **Never edit generated code manually**
- Run: `npm run db:generate` / `make db-generate`

## templ

- Input: `templates/*.templ`
- Output: `*_templ.go` (type-safe Go rendering code)
- Both `.templ` and `*_templ.go` must be committed
- Run: `templ generate` / `make templ`

## Build Order

In production: `vite build` → `go build` (Go binary reads `dist/.vite/manifest.json`)

## Source

Captured from [[sources/SRC-2026-07-06-001]].
