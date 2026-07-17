---
type: concept
status: stub
---

# HTTP Conventions

Laju Go follows strict HTTP conventions for consistency.

## POST/PUT Redirect

Use `c.Redirect(path, fiber.StatusSeeOther)` (303).

Inertia does not follow 302 correctly for form submissions — 303 is required to change POST/PUT to GET on redirect.

## PUT/PATCH Response

- **For `fetch()` calls**: Return JSON
- **For `router.put()` calls**: Redirect with 303

## Response Types

- `fiber.Map` for untyped/adhoc response data
- Typed Go structs for service boundaries and API contracts

## Session & Auth

- Sessions are database-backed (SQLite table)
- Auth middleware checks `session.Store`
- CSRF middleware only on `/app/*` routes

## CSRF Mechanism

- Axios (Inertia's HTTP client) auto-sends `XSRF-TOKEN` cookie as `X-XSRF-TOKEN` header
- Cookie is set by CSRF middleware on GET responses (`HTTPOnly: false`)
- Manual `fetch()` calls **must** include `X-XSRF-TOKEN` header — see [[concept-csrf-protection]]

## Source

Migrated from [[sources/SRC-2026-07-06-001]] (AGENTS.md).
