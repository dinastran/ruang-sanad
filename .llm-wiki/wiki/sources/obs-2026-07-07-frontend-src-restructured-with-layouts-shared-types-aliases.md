---
type: source
title: "Observation: Frontend src/ restructured with layouts + shared types + aliases"
slug: obs-2026-07-07-frontend-src-restructured-with-layouts-shared-types-aliases
status: observation
created: 2026-07-07
updated: 2026-07-07
relevance: high
observed_at: 2026-07-07T09:29:29.478Z
tags: ["frontend", "restructuring", "boilerplate", "svelte"]
---
# ⭐ Observation: Frontend src/ restructured with layouts + shared types + aliases
Restructured frontend/src/ for boilerplate standards:

Created:
- layouts/AppLayout.svelte — app shell (sidebar, mobile menu, user card) wrapping page content via Svelte 5 children snippets. Built from former Header.svelte content.
- layouts/AuthLayout.svelte — two-column auth layout (branding panel + form card) for Login/Register pages.
- components/Logo.svelte — single-source brand SVG, eliminates 5x SVG duplication across auth pages.
- lib/types.ts — shared User + Flash interfaces (replaces 3x duplicated interface User).
- lib/utils/csrf.ts — CSRF token utility (split from old helpers.ts).
- lib/notifications/toast.ts — Toast notification utility (split from old helpers.ts).

Removed:
- components/Header.svelte (content migrated to AppLayout)
- lib/utils/helpers.ts (split into csrf.ts + toast.ts)
- 5x inline Logo SVGs from Header + 4 auth pages
- 3x duplicate interface User from Header/Dashboard/Profile

Config changes:
- vite.config.js: added @lib → frontend/src/lib alias
- tsconfig.json: added baseUrl + paths for all @-aliases to fix LSP resolution

Pages updated:
- Dashboard.svelte: wraps content with AppLayout, uses shared User type
- Profile.svelte: wraps content with AppLayout, uses shared User type, uses @lib imports
- Login.svelte + Register.svelte: uses AuthLayout with branding props
- ForgotPassword.svelte + ResetPassword.svelte: uses Logo component, shared Flash type
*Relevance: high*

*Tags: frontend restructuring boilerplate svelte*
---
*Observed: 2026-07-07T09:29:29.478Z*