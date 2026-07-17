---
type: source
title: "Observation: Removed NutsDB, replaced with in-memory session cache"
slug: obs-2026-07-15-removed-nutsdb-replaced-with-in-memory-session-cache
status: observation
created: 2026-07-15
updated: 2026-07-15
relevance: high
observed_at: 2026-07-15T14:05:36.305Z
tags: ["refactor", "cache", "nutsdb", "session", "performance"]
---
# ⭐ Observation: Removed NutsDB, replaced with in-memory session cache
NutsDB has been completely removed from the project. The session cache now uses sync.RWMutex + map (in-memory).

Key changes:
- app/cache/nutsdb.go — deleted (was NutsDB wrapper with Open/Close/MMap/B-tree bucket)
- app/cache/session_cache.go — rewritten from NutsDB B-tree to in-memory sync.RWMutex + map; NewSessionCache() no longer takes parameters
- app/config/config.go — removed NutsDBPath and SessionCacheBuffer config fields
- cmd/laju-go/main.go — simplified to just cache.NewSessionCache(), no NutsDB init
- go.mod — nutsdb dependency removed via go mod tidy
- docs/guide/architecture.md, AGENTS.md, README.md — updated to remove NutsDB references
- data/cache/ — cleaned up NutsDB files (0.dat, bucket.Meta, nutsdb-flock), kept .gitkeep

Why: NutsDB as a session cache layer between Go and SQLite added ~5x complexity for marginal benefit. SQLite is the source of truth — the cache is just a performance optimization. In-memory cache is ~5-10x faster (98ns vs 605ns for GET hit) with zero disk overhead, one less dependency, and simpler initialization. Session persistence across restarts is NOT lost because SQLite remains the source of truth — the first request after restart is 5μs slower (SQLite hit) per session, which is negligible.
*Relevance: high*

*Tags: refactor cache nutsdb session performance*
---
*Observed: 2026-07-15T14:05:36.305Z*