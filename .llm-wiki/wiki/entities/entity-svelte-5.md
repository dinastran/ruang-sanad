---
type: entity
status: stub
---

# Svelte 5

**Svelte 5** is the frontend framework used in Laju Go, with runes (`$state`, `$derived`, `$effect`, `$props`) replacing the v2/v3 reactivity model.

## Role in Laju Go

- Frontend built with Svelte 5 + Inertia.js
- Entry at `frontend/src/main.ts`
- Components receive Inertia page data as props via `$props()`

## Key Rules

- **Never use `$effect` for derived state or prop initialization** — use `$derived()` or direct `$state(initial)` instead
- `$effect` is only for side effects interacting with non-reactive APIs (`document.title`, `setInterval`, `localStorage`)
- Internal navigation links must use `use:inertia` action from `@inertiajs/svelte`

## Source

Captured from [[sources/SRC-2026-07-06-001]].
