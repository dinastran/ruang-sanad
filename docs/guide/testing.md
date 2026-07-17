# Testing

This guide covers testing strategies in Laju Go.

## Overview

Laju Go supports testing at multiple levels:

| Level | Tool | Scope |
|-------|------|-------|
| **Go unit tests** | `go test` | Backend logic (services, helpers) |
| **Frontend tests** | Vitest | Svelte components, utilities |

## Go Tests

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./app/services/...
```

### Test Structure

```go
// app/services/auth_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
    hash, err := hashPassword("testpassword123")
    assert.NoError(t, err)
    assert.True(t, checkPassword(hash, "testpassword123"))
    assert.False(t, checkPassword(hash, "wrongpassword"))
}
```

### Testing Services with sqlc

Services depend on `*queries.Querier`. For testing, mock the database:

```go
func TestAuthService_Login(t *testing.T) {
    // Use an in-memory SQLite database
    db, _ := sql.Open("sqlite3", ":memory:")
    // Run migrations, create test data, test service methods
}
```

## Frontend Tests

### Running Frontend Tests

```bash
# Run all frontend tests

# Run with UI
```

### Testing with Vitest

```typescript
// frontend/src/lib/utils/helpers.test.ts
import { debounce } from './helpers';

describe('debounce', () => {
  it('should delay function execution', async () => {
    let called = false;
    const fn = debounce(() => { called = true; }, 100);
    fn();
    expect(called).toBe(false);
    await new Promise(r => setTimeout(r, 150));
    expect(called).toBe(true);
  });
});
```

## Test Best Practices

1. **Test services, not handlers** — Business logic lives in services
2. **Use in-memory SQLite** for fast, isolated DB tests
3. **Test error paths** — Invalid credentials, not found, duplicate emails
4. **Keep tests fast** — Avoid HTTP calls in unit tests

## Next Steps

- [Architecture Guide](architecture.md) — Understanding testable layers
