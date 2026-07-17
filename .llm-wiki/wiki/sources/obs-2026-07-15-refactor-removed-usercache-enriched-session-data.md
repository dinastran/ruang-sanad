---
type: source
title: "Observation: Refactor: removed UserCache, enriched session data"
slug: obs-2026-07-15-refactor-removed-usercache-enriched-session-data
status: observation
created: 2026-07-15
updated: 2026-07-15
relevance: high
observed_at: 2026-07-15T13:45:23.084Z
tags: ["refactor", "cache", "session", "backend"]
---
# ⭐ Observation: Refactor: removed UserCache, enriched session data
Removed entire UserCache layer (app/cache/user_cache.go, app/cache/user_cache_test.go) — user profile caching with NutsDB was redundant since data is now served directly from the session cache.

Key changes:
- **app/cache/user_cache.go** — deleted (was NutsDB-backed user profile cache with TTL, Get/Set/Invalidate/Clear/Size methods)
- **app/cache/user_cache_test.go** — deleted (5 tests: GetSet, GetMiss, Invalidate, Expiry, Clear, Size, Overwrite)
- **app/config/config.go** — removed UserCacheTTL field and getUserCacheTTL() helper
- **app/services/user.go** — NewUserService no longer takes *cache.UserCache; removed all cache.Invalidate() calls from UpdatePassword, UpdateAvatar, UpdateProfile, ChangePassword, DeleteAccount; removed cache reads from GetProfile and IsAdmin
- **app/session/session.go** — added Name, Avatar, EmailVerified fields to SessionData; updated CreateAuthenticatedSession signature to accept these fields; updated Save() and Regenerate() to persist them; updated all cache serialization points
- **app/cache/session_cache.go** — added Name, Avatar, EmailVerified to CachedSessionData struct
- **app/handlers/app.go** — refactored Dashboard and Profile to read user from session via sessionUser() helper instead of calling userService.GetProfile() (DB query); added toStr/toBool type-safe extractors
- **app/handlers/auth.go** — pass name, avatar, emailVerified to CreateAuthenticatedSession in Register, Login, GoogleCallback
- **app/handlers/upload.go** — sync avatar URL in session after upload
- **cmd/laju-go/main.go** — removed UserCache initialization; NewUserService called without cache arg
- **data/cache/0.dat** — binary changed (cache restructured)

Total: -410 lines, +215 lines across 10 files (excl. binary).
*Relevance: high*

*Tags: refactor cache session backend*
---
*Observed: 2026-07-15T13:45:23.084Z*