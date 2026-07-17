---
type: entity
status: stub
---

# Vite

**Vite** is the frontend build tool for Laju Go, building the Svelte 5 + Inertia.js frontend to `dist/`.

## Build Order

`vite build` must run before `go build` in production because the Go binary reads `dist/.vite/manifest.json`.

## .vite-port

Vite writes a `.vite-port` file read by Go for HMR in development. If stale, remove it: `rm .vite-port` and restart Vite.

## Source

Captured from [[sources/SRC-2026-07-06-001]].
