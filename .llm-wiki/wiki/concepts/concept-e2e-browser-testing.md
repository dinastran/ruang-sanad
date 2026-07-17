---
type: concept
status: stub
---

# E2E Browser Testing with agent_browser

Laju Go uses pi's built-in `agent_browser` for E2E testing instead of Playwright or Cypress — zero additional dependencies.

## Testing Patterns

### Auth Flow

```
open register page → fill form → submit → snapshot & verify redirect to /app
```

### Session Injection (Skip Login)

1. Insert session row in SQLite via `sqlite3` CLI
2. Set `document.cookie = 'session_id=...'` via `agent_browser eval`
3. Open protected page — already authenticated

### Unauthenticated Access

```
open /app/profile → snapshot — verify redirect to /login (Guest middleware)
```

## Advantages

- Real browser — tests actual redirects, cookies, sessions
- No mock — real Go backend + SQLite + Svelte frontend
- No extra dependencies

## Source

Captured from [[sources/SRC-2026-07-06-001]].
