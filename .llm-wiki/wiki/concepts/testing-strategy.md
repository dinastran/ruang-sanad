---
type: concept
status: stub
---

# Testing Strategy

Laju Go uses two testing approaches.

## Go Unit/Integration Tests

- **Scope**: Services, queries, handlers, cache
- **Command**: `go test ./...`
- **Setup**: In-memory SQLite, no external dependencies
- **Exception**: Test files (`*_test.go`) may call queries directly for test data setup (bypassing three-tier rule)

## E2E / Browser Tests

- **Scope**: Auth flows, form submission, page load, visual regression
- **Tool**: `agent_browser` via pi — real browser, no mock
- **Setup**: Real Go backend + SQLite + Svelte frontend
- **Dependencies**: Zero beyond pi itself

### E2E Patterns

See [[concept-e2e-browser-testing]] for detailed patterns:
- Auth flow testing (register → fill → submit → verify)
- Session injection (skip login via SQLite + cookie)
- Unauthenticated access verification (redirect to /login)

## Choosing an Approach

| Criterion | Go Test | E2E |
|-----------|---------|-----|
| Business logic | ✅ | ❌ |
| Database queries | ✅ | ❌ |
| Auth flows | ❌ | ✅ |
| Visual/redirect | ❌ | ✅ |

## Source

Migrated from [[sources/SRC-2026-07-06-001]] (AGENTS.md).
