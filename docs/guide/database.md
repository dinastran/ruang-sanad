# Database

This guide covers database setup, migrations, and query generation in Laju Go.

## Overview

Laju Go uses **SQLite** as the database with **sqlc** for type-safe query generation and **Goose** for migrations. This combination provides:

- **Zero configuration** — No database server to manage
- **Type-safe queries** — sqlc generates Go code from SQL at compile time
- **Version control** — Goose manages schema migrations
- **Production-ready** — SQLite with WAL mode and connection pooling

## Database Setup

### Connection Initialization

```go
// cmd/laju-go/main.go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"  // CGO-based SQLite driver
)

func initDatabase(dbPath string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }

    // Configure connection pool (optimized for SQLite single-instance)
    db.SetMaxOpenConns(15)                  // Maximum open connections
    db.SetMaxIdleConns(10)                  // Keep idle connections ready
    db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections
    db.SetConnMaxIdleTime(30 * time.Second) // Recycle stale idle connections

    // Apply production optimizations
    applySQLiteOptimizations(db)

    return db, nil
}
```

### SQLite Optimizations

```go
func applySQLiteOptimizations(db *sql.DB) {
    pragmas := []string{
        "PRAGMA journal_mode = WAL",                  // Write-Ahead Logging
        "PRAGMA synchronous = NORMAL",                 // Balance speed/durability
        "PRAGMA cache_size = -16000",                  // 16MB cache (KB, neg=kilobytes)
        "PRAGMA mmap_size = 268435456",                // 256MB memory-mapped I/O
        "PRAGMA temp_store = MEMORY",                  // Memory temp tables
        "PRAGMA busy_timeout = 5000",                  // 5 second lock wait
        "PRAGMA foreign_keys = ON",                    // Enable foreign keys
        "PRAGMA wal_autocheckpoint = 1000",            // WAL checkpoint pages
    }

    for _, pragma := range pragmas {
        db.Exec(pragma)
    }
}
```

### Why These Settings?

| Setting | Value | Benefit |
|---------|-------|---------|
| `journal_mode` | WAL | Better write concurrency, readers don't block writers |
| `synchronous` | NORMAL | Safe for WAL mode, faster than FULL |
| `cache_size` | 16MB | Reduces disk I/O for frequent queries |
| `mmap_size` | 256MB | NVMe memory-mapped I/O for faster reads |
| `temp_store` | MEMORY | Faster temporary table operations |
| `busy_timeout` | 5000ms | Automatic retry on database locks |
| `foreign_keys` | ON | Enforce referential integrity |

### Connection Pool Settings

| Setting | Value | Why |
|---------|-------|-----|
| `MaxOpenConns` | 15 | SQLite is single-writer; more connections don't help |
| `MaxIdleConns` | 10 | Keep connections warm, avoid reconnect overhead |
| `ConnMaxLifetime` | 5 min | Recycle connections periodically |
| `ConnMaxIdleTime` | 30 sec | Free stale connections faster |

### Why mattn/go-sqlite3 (CGO-enabled)?

The project uses `github.com/mattn/go-sqlite3` — a **CGO-based** driver that delivers ~1.3–1.9x higher throughput than pure-Go alternatives. The trade-off is worth it for production workloads:

| Factor | `mattn/go-sqlite3` ✅ |
|--------|------------------------|
| Performance | 2x throughput vs pure-Go drivers (100K+ RPS on $24/mo Vultr) |
| Cross-compile | `make build-linux` (requires `brew install zig` for `zig cc`) |
| Static binary | Yes — `zig cc` links libsqlite3 statically |
| Dev setup | Works out of the box on macOS (CGO enabled by default) |

> **Driver name**: Use `sql.Open("sqlite3", dbPath)` — mattn registers as `"sqlite3"`, unlike modernc which registers as `"sqlite"`.

## Database Migrations with Goose

### Creating Migrations

```bash
# Via npm scripts
npm run db:migrate:create add_users_table
```

Or directly with goose:

```bash
goose -dir migrations create add_users_table sqlite3
# Output: migrations/20240101120000_add_users_table.sql
```

### Migration File Structure

```sql
-- migrations/0001_create_users_table.sql
-- +goose Up
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password TEXT,
    avatar TEXT DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    google_id TEXT UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);

-- +goose Down
DROP TABLE IF EXISTS users;
```

### Running Migrations

```bash
# Run all pending migrations
npm run db:migrate

# Check migration status
npm run db:migrate:status

# Rollback last migration
npm run db:migrate:down

# Reset database (delete + recreate)
npm run db:refresh

# Via goose directly
goose -dir migrations sqlite3 data/app.db up
```

### Auto-Run Migrations on Startup

Migrations run **automatically** when the application starts:

```go
func runMigrations(db *sql.DB, migrationsDir string) error {
    goose.SetBaseFS(nil)
    if err := goose.SetDialect("sqlite"); err != nil {
        return err
    }
    return goose.Up(db, migrationsDir)
}
```

## Query Generation with sqlc

Laju Go uses **[sqlc](https://sqlc.dev/)** to generate type-safe Go code from SQL. Instead of hand-writing repositories or using query builders, you write SQL and sqlc generates the Go code.

### Workflow

```
1. Write SQL → queries/user.sql
2. Generate  → npm run db:generate
3. Use code  → s.querier.GetUserByEmail(ctx, email)
```

### Why sqlc Instead of Squirrel/ORM?

| Approach | Runtime Safety | Performance | Boilerplate |
|----------|---------------|-------------|-------------|
| **sqlc** | Compile-time checked | Native SQL | Zero (generated) |
| Squirrel | Runtime errors | SQL building overhead | Manual per query |
| GORM | Reflection bugs | Slow (reflect) | Minimal |

### Writing SQL Queries

Write queries in `queries/*.sql`:

```sql
-- queries/user.sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: CreateUser :execresult
INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, ?);

-- name: UpdateUser :exec
UPDATE users SET name = ?, avatar = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE users SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
```

The `:one`, `:many`, `:exec`, `:execresult` annotations tell sqlc what the query returns.

### Generating Go Code

```bash
npm run db:generate
# Or: sqlc generate
```

This generates:

```
app/queries/
├── db.go                    # Transaction helpers
├── models.go                # Go structs matching table schemas
├── querier.go               # Querier wrapper (what services use)
├── user.sql.go              # Generated user query methods
├── session.sql.go           # Generated session query methods
└── session_helpers.go       # Helper functions
```

### Using Generated Queries in Services

```go
type AuthService struct {
    querier *queries.Querier
}

func (s *AuthService) Login(email, password string) (*models.User, error) {
    user, err := s.querier.GetUserByEmail(context.Background(), email)
    if err != nil {
        if errors.Is(err, queries.ErrUserNotFound) {
            return nil, ErrInvalidCredentials
        }
        return nil, err
    }
    // ... validate password ...
    return user, nil
}
```

### Querier Wrapper Pattern

The `app/queries/querier.go` wraps sqlc's generated `*Queries` to add convenience methods:

```go
type Querier struct {
    *Queries  // sqlc-generated methods embedded
}

func NewQuerier(db DBTX) *Querier {
    return &Querier{Queries: New(db)}
}

// GetUserByEmail wraps sqlc with error handling + model mapping
func (q *Querier) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    qUser, err := q.Queries.GetUserByEmail(ctx, email)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    return toModelUser(qUser), nil
}
```

### Complete CRUD Example

**SQL source** (`queries/user.sql`):

```sql
-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = ?;

-- name: CreateUser :execresult
INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, ?);
```

**Usage in service** (`app/services/auth.go`):

```go
func (s *AuthService) Register(name, email, password string) (*models.User, error) {
    // Check if user already exists
    _, err := s.querier.GetUserByEmail(context.Background(), email)
    if err == nil {
        return nil, queries.ErrUserAlreadyExists
    }

    // Hash password
    hashedPassword, _ := hashPassword(password)

    // Create user
    user := &models.User{
        Email: email,
        Name:  name,
        Password: sql.NullString{String: hashedPassword, Valid: true},
        Role:  models.RoleUser,
    }

    if err := s.querier.CreateUser(context.Background(), user); err != nil {
        return nil, err
    }

    return user, nil
}
```

### Error Handling with sqlc

sqlc-generated code uses `sql.ErrNoRows` for missing results. The Querier wrapper converts these to domain-specific errors:

```go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrUserAlreadyExists = errors.New("user already exists")
)

func (q *Querier) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    qUser, err := q.Queries.GetUserByEmail(ctx, email)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    return toModelUser(qUser), nil
}
```

### Adding a New Query

1. Add SQL to `queries/*.sql`:

```sql
-- name: GetUserCount :one
SELECT COUNT(*) FROM users;
```

1. Regenerate:

```bash
npm run db:generate
```

1. Use in service:

```go
count, err := s.querier.GetUserCount(ctx)
```

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password TEXT,
    avatar TEXT DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    google_id TEXT UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);
```

### Sessions Table

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    data TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

## Transactions

Use `db.go`'s transaction helpers for atomic operations:

```go
// app/queries/db.go provides BeginTx helper
tx, err := querier.BeginTx(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

// Execute queries within transaction
tx.Queries.UpdateUser(ctx, user)
tx.Queries.DeleteSession(ctx, sessionID)

// Commit
return tx.Commit()
```

> Transactions in sqlc can also be done directly with `database/sql`:
>
> ```go
> tx, _ := db.Begin()
> defer tx.Rollback()
> tx.Exec("UPDATE users SET ... WHERE id = ?", id)
> return tx.Commit()
> ```

## Best Practices

### 1. Use sqlc Instead of Raw SQL

All database operations should go through the generated `Querier`. Avoid raw SQL in services:

```go
// ❌ Bad: Raw SQL in service
db.Exec("SELECT * FROM users WHERE email = ?", email)

// ✅ Good: Generated type-safe method
user, err := s.querier.GetUserByEmail(ctx, email)
```

### 2. Handle sql.ErrNoRows with Domain Errors

```go
// The Querier wrapper already converts sql.ErrNoRows to domain errors:
if errors.Is(err, queries.ErrUserNotFound) {
    return nil, services.ErrInvalidCredentials
}
```

### 3. Use Transactions for Multiple Writes

```go
// ✅ Good: Transaction for data integrity
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()

// Multiple operations
tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID)
tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID)

return tx.Commit()
```

### 4. Index Frequently Queried Columns

```sql
-- Already indexed in migrations
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

### 5. Connection Pooling (Already Configured)

The pool settings in `initDatabase()` are tuned for SQLite. No changes needed.

### 6. Close Rows After Query

When using raw `database/sql` directly:

```go
rows, err := db.Query("SELECT ...")
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    // Scan row
}
```

> With sqlc, rows are closed internally — no need to manage manually.

## Troubleshooting

### Database Locked

**Problem**: `database is locked`

**Solutions**:

1. Enable WAL mode: `PRAGMA journal_mode=WAL`
2. Set busy timeout: `PRAGMA busy_timeout=5000`
3. Reduce concurrent writes
4. Check for unclosed transactions

### Migration Failed

**Problem**: Migration fails on startup

**Solutions**:

1. Check migration syntax
2. Verify database path
3. Run migrations manually: `goose -dir migrations sqlite3 data/app.db up`
4. Check goose_db_version table

### Connection Issues

**Problem**: `unable to open database file`

**Solutions**:

1. Ensure directory exists: `mkdir -p data`
2. Check permissions: `chmod 755 data`
3. Verify DB_PATH in .env

## Next Steps

- [Authentication Guide](authentication.md) - User authentication and sessions
- [Architecture Guide](architecture.md) - sqlc and Querier pattern in context
- [Deployment Guide](../deployment/production.md) - Production database setup
