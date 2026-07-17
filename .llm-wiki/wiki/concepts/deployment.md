---
type: concept
status: stub
---

# Deployment

Laju Go supports three deployment methods.

## Git-Based Deployment

```bash
git pull
make build
sudo systemctl restart laju-go
```

## Docker

Multi-stage build in `Dockerfile`. Build with `docker build .` / `make docker`.

## Systemd

Service unit file at `systemd/laju-go.service`.

## Cross-Compilation

Building for Linux from macOS requires CGO cross-compilation:

```bash
brew install zig
make build-linux
```

Uses `zig cc` as a cross-compiler for CGO. This is needed because `github.com/mattn/go-sqlite3` is CGO-based.

## Build Order

In production: `vite build` must run **before** `go build` — the Go binary reads `dist/.vite/manifest.json`.

## Source

Migrated from [[sources/SRC-2026-07-06-001]] (AGENTS.md).
