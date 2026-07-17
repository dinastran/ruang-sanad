---
type: concept
status: stub
---

# Inertia SPA Navigation

Inertia.js enables single-page application (SPA) navigation in Laju Go without building a separate API. Server-side integration via [fiber-inertia](https://github.com/maulanashalihin/fiber-inertia) — native `func(c *fiber.Ctx) error`, published Go library.

## Navigation Types

| Action | Method | Notes |
|--------|--------|-------|
| Internal links | `<a href="/path" use:inertia>` | Uses `use:inertia` action from `@inertiajs/svelte` |
| OAuth links | `<a href="/auth/google">` | Plain `<a>` — must redirect to external provider |
| Form submission | `router.post()` / `router.put()` | From `@inertiajs/svelte` |
| File upload | `fetch()` + FormData | Must include CSRF header, then `router.put()` to save URL |

## Redirect Rules

- POST/PUT redirect: `h.inertiaService.Redirect(c, path)` — returns 303 See Other, Inertia client follows via XHR
- External redirect (OAuth, external URLs): `h.inertiaService.Location(c, url)` — returns 409 + `X-Inertia-Location`, triggers `window.location`
- Back navigation: `h.inertiaService.Back(c)` or `h.inertiaService.Back(c, "/fallback")` — reads Referer header
- Inertia does not follow 302 correctly for form submissions — use `Redirect()` (303) not raw `c.Redirect()`

## Library

Inertia server-side logic handled by `github.com/maulanashalihin/fiber-inertia`:

- `app/services/inertia.go` wraps the library and adds laju-go-specific features (Vite URLs, CSRF injection, flash messages)
- All methods promoted via embedding: `Render`, `Redirect`, `Location`, `Back`, `Middleware`
- See [fiber-inertia docs](https://github.com/maulanashalihin/fiber-inertia) for full API

## Source

Captured from [[sources/obs-2026-07-16-fiber-inertia-integrated-into-laju-go]] — integration of fiber-inertia library.
