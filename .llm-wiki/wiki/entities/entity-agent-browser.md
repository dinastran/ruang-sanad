---
type: entity
status: stub
---

# agent_browser (Pi)

**agent_browser** is pi's built-in browser testing tool used for E2E testing in Laju Go. No Playwright/Cypress dependencies.

## Usage in Laju Go

- Auth flow testing (register, login, redirects)
- Session injection for testing protected pages without login
- Visual verification via screenshots
- Page load and redirect assertion

## Key Patterns

- **Form submission test**: `open` → `fill` → `click` → `snapshot`
- **Session injection**: Insert session row in SQLite via `sqlite3` CLI, then set `document.cookie` via `eval`
- **Unauthenticated access test**: Open protected route → verify redirect to `/login`

## Source

Captured from [[sources/SRC-2026-07-06-001]].
