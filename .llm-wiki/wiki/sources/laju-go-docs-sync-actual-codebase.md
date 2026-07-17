---
type: source
title: "Docs Laju Go diselaraskan dengan actual codebase"
slug: laju-go-docs-sync-actual-codebase
status: insight
created: 2026-07-06
updated: 2026-07-06
category: maintenance
---
# Docs Laju Go diselaraskan dengan actual codebase
Audit docs/ Laju Go menemukan 18 file docs, 10 di antaranya butuh perbaikan karena tidak mencerminkan actual codebase. Perbaikan meliputi: [[concept-cgo-cross-compilation]] (fix "zero CGO" claim ke mattn/go-sqlite3 CGO), [[entity-go-fiber]] route signatures, [[entity-sqlite]] MailerService constructor, handler constructor signatures, dan referensi ke file yang sudah tidak ada di docs/README.md. Landing page template juga diperbaiki dari klaim modernc ke mattn yang sebenarnya dipakai di production. Git-based deployment (clone → build → systemd restart) ditambahkan sebagai deployment strategy utama.
*Category: maintenance*
---
*Captured: 2026-07-06*
## Related
_Add links to related pages._