---
type: source
title: "Observation: CSRF middleware refactored to double-submit cookie pattern"
slug: obs-2026-07-17-csrf-middleware-refactored-to-double-submit-cookie-pattern
status: observation
created: 2026-07-17
updated: 2026-07-17
relevance: high
observed_at: 2026-07-17T03:58:47.848Z
tags: ["csrf", "security", "refactor", "middleware", "double-submit-cookie"]
source_context: "Reviewing git status and git diff — found CSRF middleware refactor to double-submit cookie pattern"
---
# ⭐ Observation: CSRF middleware refactored to double-submit cookie pattern
Refactored CSRF middleware from session-based to double-submit cookie pattern. Changes across 3 files:

- `app/middlewares/csrf.go`: Removed dependency on `session.Store`, `slog`. Token now stored ONLY in cookie (XSRF-TOKEN). Validation compares X-XSRF-TOKEN header with XSRF-TOKEN cookie value. Removed `isTokenExpired()`, simplified `setToken()` and `validateToken()`. Reduced from ~100 lines to ~35 lines.
- `routes/web.go`: `SetupCSRFMiddleware` signature no longer takes `*session.Store`.
- `cmd/laju-go/main.go`: Caller updated accordingly — `sessionStore` no longer passed.

Benefit: faster validation (zero session I/O), stateless, simpler code, more appropriate for SPA/Inertia.js architecture.
*Relevance: high*

*Context: Reviewing git status and git diff — found CSRF middleware refactor to double-submit cookie pattern*

*Tags: csrf security refactor middleware double-submit-cookie*
---
*Observed: 2026-07-17T03:58:47.848Z*