---
type: entity
status: stub
---

# Inertia.js

**Inertia.js** is the glue layer connecting the Go backend (Fiber) with Svelte 5 frontend in Laju Go. It enables SPA navigation without an API.

## How It Works

- **Initial page load**: Server renders `templates.InertiaPage` HTML shell with JSON page data
- **Subsequent navigation**: XHR with `X-Inertia: true` header → server returns JSON `{component, props, url}`
- Handlers use `inertiaService.Render(c, "ComponentName", fiber.Map{...})` — never render HTML directly

## Key Conventions

- Internal links must use `use:inertia` action (imported from `@inertiajs/svelte`) for SPA navigation
- OAuth links (`/auth/google`, `/auth/github`) use plain `<a>` without `use:inertia`
- Form submissions use `router.post()` / `router.put()` from `@inertiajs/svelte`
- POST/PUT redirects must use HTTP 303 (`StatusSeeOther`)
- File uploads require manual `fetch()` with FormData (Inertia cannot send binary)

## Source

Captured from [[sources/SRC-2026-07-06-001]].
