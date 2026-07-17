---
type: entity
status: stub
---

# Go Fiber

**Go Fiber** is the web framework used in Laju Go as the HTTP server. It provides a Express.js-like API for Go, used for routing, middleware, and request/response handling.

## Role in Laju Go

- Single entry point at `cmd/laju-go/main.go`
- Route definitions in `routes/web.go`
- Request parsing and response rendering in `app/handlers/`
- Middleware stack includes session, CSRF, guest/auth, and Inertia middleware
- Uses `c.Redirect(path, fiber.StatusSeeOther)` (303) for Inertia-compatible form redirects

## Key Conventions

- `fiber.Map` for untyped response data in handlers
- CSRF middleware only on `/app/*` routes
- Sessions are database-backed (SQLite), checked via middleware

## Source

Captured from [[sources/SRC-2026-07-06-001]].
