---
type: concept
status: stub
---

# CGO Cross-Compilation

Laju Go uses `github.com/mattn/go-sqlite3` (CGO-based), requiring CGO for cross-compilation.

## Cross-Compile to Linux from macOS

1. Install Zig: `brew install zig`
2. Run: `make build-linux`

The build uses `zig cc` as a cross-compiler, which handles CGO cross-compilation transparently.

## Source

Captured from [[sources/SRC-2026-07-06-001]].
