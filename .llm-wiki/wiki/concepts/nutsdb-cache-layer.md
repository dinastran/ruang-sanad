# NutsDB Cache Layer (REMOVED)

> ⚠️ **NutsDB has been removed as of commit `a9e8001`.**
> Session cache now uses in-memory `sync.RWMutex` + `map`.
> See `app/cache/session_cache.go` and [[sources/obs-2026-07-15-removed-nutsdb-replaced-with-in-memory-session-cache]] for details.

## Overview (Historical)

Laju Go uses [NutsDB](https://github.com/nutsdb/nutsdb) — an embedded key-value store written in pure Go — as a **persistent TTL cache** for sessions and user profiles. Unlike in-memory caches, NutsDB data survives server restarts with no data loss.

## Architecture

### Wrapper (`app/cache/nutsdb.go`)

Shared NutsDB instance opened at startup with:

- **MMap** RW mode (memory-mapped files for near-RAM read speed)
- **HintKeyValAndRAMIdxMode** (index in RAM for fast lookups)
- **16MB segments**
- Auto-creates buckets: `"sessions"` and `"users"`

### Buckets

| Bucket | Key | Value | TTL Strategy |
|--------|-----|-------|-------------|
| `"sessions"` | Session ID (string) | `CachedSessionData` (JSON) | Remaining session TTL + buffer |
| `"users"` | User ID (int64 big-endian) | `userCacheEntry` (JSON + app-level ExpiresAt) | `USER_CACHE_TTL` env var (default 30m) |

### Dual TTL Safety

Each entry has two layers of expiry:

1. **Application-level** `ExpiresAt` field in the struct — sub-second precision
2. **NutsDB native TTL** on `tx.Put()` — second granularity, acts as backup safety net

If app-level TTL expires, the entry is lazily invalidated on next `Get()`. The NutsDB native TTL ensures the entry is eventually cleaned up by the database engine.

## Cache Types

### UserCache (`app/cache/user_cache.go`)

- Key: `int64Key(userID)` (8 bytes big-endian)
- TTL: `USER_CACHE_TTL` (default 30m, set 0 to disable)
- Methods: `Get()`, `Set()`, `Invalidate()`, `Clear()`, `Size()`
- Used by `UserService` for profile lookups and role checks
- Auto-invalidated on profile updates

### SessionCache (`app/cache/session_cache.go`)

- Key: `sessionID` (string, raw bytes)
- TTL: Remaining session lifetime + `SESSION_CACHE_BUFFER` buffer (default 5m)
- Methods: `Get()`, `Set()`, `Invalidate()`, `Clear()`
- Stores: UserID, Email, Role, CSRFToken, IP, UserAgent, ExpiresAt
- Used by `session.Store` to reduce DB reads on every authenticated request

## Wiring (`cmd/laju-go/main.go`)

```go
ndb, _ := cache.Open(cfg.NutsDBPath)            // default: ./data/cache
userCache := cache.NewUserCache(ndb.DB, cfg.UserCacheTTL)
sessionCache := cache.NewSessionCache(ndb.DB, cfg.SessionCacheBuffer)
sessionStore := session.New(querier, sessionCache, cfg.SessionTTL)
userService := services.NewUserService(querier, userCache)
```

## Configuration

| Env | Default | Description |
|-----|---------|-------------|
| `NUTSDB_PATH` | `./data/cache` | NutsDB data directory |
| `USER_CACHE_TTL` | `30m` | User profile cache TTL |
| `SESSION_CACHE_BUFFER` | `5m` | Buffer added to remaining session lifetime for NutsDB TTL |

## Benefits

- ✅ Cache survives server restarts (no mass logout after deploy)
- ✅ Thread-safe via NutsDB transaction isolation (no `sync.RWMutex`)
- ✅ Auto TTL cleanup + explicit invalidation
- ✅ Near-RAM read performance via MMap
- ✅ Zero data loss on restart

## References

- Source: `app/cache/nutsdb.go`, `app/cache/session_cache.go`, `app/cache/user_cache.go`
- Commit: `70d42c6`
- [[sources/obs-2026-07-15-nutsdb-persistent-cache-layer-added]]
