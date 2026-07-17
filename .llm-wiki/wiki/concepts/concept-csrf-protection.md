---
type: concept
status: stub
---

# CSRF Protection

Laju Go uses CSRF middleware on `/app/*` routes. CSRF tokens are set via cookie on GET responses (`HTTPOnly: false`).

## How It Works

- **Axios/Inertia**: Auto-sends `XSRF-TOKEN` cookie as `X-XSRF-TOKEN` header — no manual handling needed
- **Manual `fetch()`**: **Must** include `X-XSRF-TOKEN` header read from cookie via `getCSRFToken()` helper from `$lib/utils/helpers`

## Critical Rule

Every `fetch()` to CSRF-protected routes (`/app/*`, `/admin/*`) **must** include:
```typescript
headers: { "X-XSRF-TOKEN": getCSRFToken() }
```

Without it, the request is rejected with 400 "CSRF token missing".

## Source

Captured from [[sources/SRC-2026-07-06-001]].
