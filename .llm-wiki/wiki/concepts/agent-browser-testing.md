---
type: concept
status: stub
---

# Agent Browser Testing

Laju Go menggunakan pi's built-in `agent_browser` untuk E2E testing — tanpa Playwright/Cypress.

## Skip Login — Inject Session Langsung

Untuk test halaman protected tanpa perlu login manual:

1. Buat session di SQLite
2. Set cookie via `agent_browser eval`
3. Buka halaman protected

Detail lengkap: [[concept-e2e-browser-testing]]

## Source

Migrated from AGENTS.md.
