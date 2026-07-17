---
type: source
title: "Observation: NutsDB persistent cache layer added"
slug: obs-2026-07-15-nutsdb-persistent-cache-layer-added
status: observation
created: 2026-07-15
updated: 2026-07-15
relevance: high
observed_at: 2026-07-15T07:13:21.582Z
tags: ["cache", "nutsdb", "architecture", "persistent-storage"]
source_context: "Explaining commit 70d42c6 and updating docs"
---
# ⭐ Observation: NutsDB persistent cache layer added
Commit 70d42c6 replaced in-memory sync.RWMutex+map cache with NutsDB-backed persistent cache. Files affected: app/cache/nutsdb.go (new wrapper), app/cache/session_cache.go (refactored to NutsDB), app/cache/user_cache.go (refactored to NutsDB), app/cache/user_cache_test.go (tests with temp NutsDB), app/config/config.go (added NutsDBPath, UserCacheTTL, SessionCacheTTL), cmd/laju-go/main.go (wiring). Uses MMap mode for near-RAM speed, dual TTL safety (app-level + NutsDB native). Path defaults to ./data/cache. Docs updated: README.md and docs/guide/architecture.md section 9.
*Relevance: high*

*Context: Explaining commit 70d42c6 and updating docs*

*Tags: cache nutsdb architecture persistent-storage*
---
*Observed: 2026-07-15T07:13:21.582Z*