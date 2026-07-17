# SQLite Configuration Guide

Complete guide for SQLite configuration in Laju Go based on server resources and production use cases.

## Overview

Laju Go uses SQLite with optimizations that can be customized based on:
- **Available RAM** - Determines cache size and connection pool
- **CPU cores** - Determines concurrent connections
- **Storage type** - NVMe vs SSD vs HDD (mmap optimization)
- **Traffic pattern** - Read-heavy vs write-heavy workloads

## Current Configuration (Default)

The default configuration in `main.go` is optimized for **Vultr High Frequency 1-2GB RAM**:

```go
// cmd/laju-go/main.go - initDatabase()
db.SetMaxOpenConns(15)                    // Connection pool size
db.SetMaxIdleConns(10)                    // Keep idle connections ready
db.SetConnMaxLifetime(5 * time.Minute)    // Connection lifetime

db.Exec("PRAGMA journal_mode = WAL")           // Write-Ahead Logging
db.Exec("PRAGMA synchronous = NORMAL")         // Balance speed/durability
db.Exec("PRAGMA cache_size = -16000")          // 16MB cache
db.Exec("PRAGMA mmap_size = 268435456")        // 256MB mmap
db.Exec("PRAGMA temp_store = MEMORY")          // Memory temp tables
db.Exec("PRAGMA busy_timeout = 5000")          // 5 second lock wait
db.Exec("PRAGMA wal_autocheckpoint = 1000")    // Checkpoint frequency
```

## Configuration by RAM Size

### 1. Small Server (512MB - 1GB RAM)

**Use case**: Development, low-traffic staging, MVP

```go
// Connection pooling
db.SetMaxOpenConns(10)                    // Conservative pool
db.SetMaxIdleConns(3)                     // Minimal idle
db.SetConnMaxLifetime(5 * time.Minute)

// SQLite PRAGMAs
db.Exec("PRAGMA cache_size = -8000")           // 8MB cache (~1.5% of 512MB)
db.Exec("PRAGMA mmap_size = 134217728")        // 128MB mmap (virtual)
db.Exec("PRAGMA busy_timeout = 5000")          // 5 second timeout
db.Exec("PRAGMA wal_autocheckpoint = 1000")    // Default checkpoint
db.Exec("PRAGMA synchronous = NORMAL")         // Safe for WAL
db.Exec("PRAGMA journal_mode = WAL")           // WAL mode
db.Exec("PRAGMA temp_store = MEMORY")          // Memory temp tables
```

**Memory breakdown**:
```
App (Go/Fiber)      : ~200-300MB
SQLite cache        : 8MB
SQLite mmap (virt)  : 128MB (not physical RAM)
OS overhead         : ~150MB
Buffer/Headroom     : ~50-100MB
─────────────────────────────────
Total               : ~400-500MB of 512MB (80-95% usage)
```

**⚠️ Warning**: For 512MB, **you must add swap file** at least 512MB:
```bash
# Create swap file
sudo fallocate -l 512M /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make permanent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

---

### 2. Medium Server (2GB - 4GB RAM) ✅ **RECOMMENDED**

**Use case**: Production apps, medium traffic (10k-50k users/day)

```go
// Connection pooling
db.SetMaxOpenConns(25)                    // Moderate pool
db.SetMaxIdleConns(10)                    // Keep connections warm
db.SetConnMaxLifetime(5 * time.Minute)

// SQLite PRAGMAs
db.Exec("PRAGMA cache_size = -32000")          // 32MB cache (~1% of 4GB)
db.Exec("PRAGMA mmap_size = 536870912")        // 512MB mmap
db.Exec("PRAGMA busy_timeout = 7500")          // 7.5 second timeout
db.Exec("PRAGMA wal_autocheckpoint = 2000")    // Less frequent checkpoint
db.Exec("PRAGMA synchronous = NORMAL")         // Safe for WAL
db.Exec("PRAGMA journal_mode = WAL")           // WAL mode
db.Exec("PRAGMA temp_store = MEMORY")          // Memory temp tables
```

**Memory breakdown**:
```
App (Go/Fiber)      : ~300-400MB
SQLite cache        : 32MB
SQLite mmap (virt)  : 512MB (virtual)
OS overhead         : ~200MB
Buffer/Headroom     : ~1GB+
─────────────────────────────────
Total               : ~600-700MB of 2-4GB (20-35% usage)
```

**Expected performance**:
- Max RPS: ~15,000-25,000
- Concurrent users: 500-1,000
- Cache hit ratio: ~85-90%
- P99 latency: ~30-50ms

---

### 3. Large Server (8GB - 16GB RAM)

**Use case**: High-traffic production (50k-200k users/day)

```go
// Connection pooling
db.SetMaxOpenConns(50)                    // Large pool
db.SetMaxIdleConns(15)                    // More idle connections
db.SetConnMaxLifetime(10 * time.Minute)   // Longer lifetime

// SQLite PRAGMAs
db.Exec("PRAGMA cache_size = -500000")         // 500MB cache (~3% of 16GB)
db.Exec("PRAGMA mmap_size = 1073741824")       // 1GB mmap
db.Exec("PRAGMA busy_timeout = 10000")         // 10 second timeout
db.Exec("PRAGMA wal_autocheckpoint = 3000")    // Less frequent checkpoint
db.Exec("PRAGMA synchronous = NORMAL")         // Safe for WAL
db.Exec("PRAGMA journal_mode = WAL")           // WAL mode
db.Exec("PRAGMA temp_store = MEMORY")          // Memory temp tables
```

**Memory breakdown**:
```
App (Go/Fiber)      : ~400-600MB
SQLite cache        : 500MB
SQLite mmap (virt)  : 1GB (virtual)
OS overhead         : ~300MB
Buffer/Headroom     : ~6-10GB
─────────────────────────────────
Total               : ~1.5-2GB of 8-16GB (10-25% usage)
```

**Expected performance**:
- Max RPS: ~40,000-60,000
- Concurrent users: 2,000-3,000
- Cache hit ratio: ~92-95%
- P99 latency: ~15-25ms

---

### 4. Enterprise Server (32GB - 64GB RAM) 🚀

**Use case**: Enterprise production (200k-1M+ users/day)

```go
// Connection pooling
db.SetMaxOpenConns(100)                   // Maximum pool for 16+ vCPU
db.SetMaxIdleConns(25)                    // Many idle connections
db.SetConnMaxLifetime(10 * time.Minute)   // Long lifetime

// SQLite PRAGMAs
db.Exec("PRAGMA cache_size = -4000000")        // 4GB cache (~7% of 58GB)
db.Exec("PRAGMA mmap_size = 2147483648")       // 2GB mmap (NVMe optimization)
db.Exec("PRAGMA busy_timeout = 10000")         // 10 second timeout
db.Exec("PRAGMA wal_autocheckpoint = 5000")    // Minimal checkpoint overhead
db.Exec("PRAGMA synchronous = NORMAL")         // Safe for WAL
db.Exec("PRAGMA journal_mode = WAL")           // WAL mode
db.Exec("PRAGMA temp_store = MEMORY")          // Memory temp tables
db.Exec("PRAGMA page_size = 4096")             // Explicit page size
```

**Memory breakdown** (example: 58GB RAM):
```
App (Go/Fiber)      : ~500-800MB
SQLite cache        : 4GB
SQLite mmap (virt)  : 2GB (virtual)
OS overhead         : ~500MB
Buffer/Headroom     : ~50GB+
─────────────────────────────────
Total               : ~5-6GB of 58GB (10% usage)
```

**Expected performance**:
- Max RPS: ~80,000-120,000
- Concurrent users: 5,000-10,000
- Cache hit ratio: ~96-99%
- P99 latency: ~8-15ms

**⚠️ Important**: SQLite has inherent limitations:
- **Max concurrent writes**: 1 at a time (WAL helps, but still limited)
- **Write contention**: Starts after ~50-100 concurrent connections

For write-heavy workloads with this hardware, **consider PostgreSQL**.

---

## Configuration by Workload

### Read-Heavy Workload (>90% reads)

**Optimization for maximum caching**:

```go
// Increase cache size
db.Exec("PRAGMA cache_size = -1000000")  // 1GB cache
db.Exec("PRAGMA mmap_size = 2147483648") // 2GB mmap

// Aggressive connection pooling
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(25)
```

**Reason**: More data cached = less disk I/O

---

### Write-Heavy Workload (>30% writes)

**Optimization for write throughput**:

```go
// Reduce checkpoint frequency
db.Exec("PRAGMA wal_autocheckpoint = 10000")  // Less frequent checkpoints

// Keep cache moderate (WAL needs memory too)
db.Exec("PRAGMA cache_size = -32000")  // 32MB cache

// Increase busy timeout for write contention
db.Exec("PRAGMA busy_timeout = 15000")  // 15 second timeout
```

**Reason**: Less frequent checkpoints = better write performance

**⚠️ Warning**: SQLite is not suitable for >50% write workloads. Consider PostgreSQL.

---

### Mixed Workload (70% reads, 30% writes)

**Balanced configuration** (default is sufficient):

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.Exec("PRAGMA cache_size = -32000")       // 32MB
db.Exec("PRAGMA wal_autocheckpoint = 2000") // Balanced
```

---

## PRAGMA Reference

### Connection Pooling

| Setting | Recommended Range | Description |
|---------|-------------------|-------------|
| `MaxOpenConns` | 10-100 | Max concurrent DB connections |
| `MaxIdleConns` | 3-25 | Idle connections to keep |
| `ConnMaxLifetime` | 5-10 min | Connection reuse duration |

**Rule of thumb**: `MaxOpenConns = 6-8 × CPU cores`

---

### Cache Size

```sql
PRAGMA cache_size = -<KB>  -- Negative value = KB
```

| RAM | Cache Size | Value |
|-----|------------|-------|
| 512MB | 8MB | `-8000` |
| 1GB | 16MB | `-16000` |
| 2GB | 32MB | `-32000` |
| 4GB | 64MB | `-64000` |
| 8GB | 256MB | `-256000` |
| 16GB | 500MB | `-500000` |
| 32GB | 2GB | `-2000000` |
| 58GB | 4GB | `-4000000` |

**Rule of thumb**: 1-7% of total RAM

---

### Mmap Size

```sql
PRAGMA mmap_size = <bytes>
```

| Storage Type | Mmap Size | Value |
|--------------|-----------|-------|
| HDD | 64MB | `67108864` |
| SSD | 256MB | `268435456` |
| NVMe | 1-2GB | `1073741824` - `2147483648` |

**Reason**: NVMe benefits greatly from memory-mapped I/O

---

### Busy Timeout

```sql
PRAGMA busy_timeout = <milliseconds>
```

| Concurrency | Timeout | Value |
|-------------|---------|-------|
| Low (<100 RPS) | 3 seconds | `3000` |
| Medium (100-1000 RPS) | 5-7 seconds | `5000-7500` |
| High (>1000 RPS) | 10 seconds | `10000` |
| Very High (write contention) | 15 seconds | `15000` |

---

### WAL Autocheckpoint

```sql
PRAGMA wal_autocheckpoint = <pages>
```

| Workload | Checkpoint | Value |
|----------|------------|-------|
| Read-heavy | 1000 pages | `1000` (default) |
| Mixed | 2000-3000 pages | `2000-3000` |
| Write-heavy | 5000-10000 pages | `5000-10000` |

**Trade-off**: Less frequent = better write perf, more disk usage

---

### Synchronous

```sql
PRAGMA synchronous = FULL | NORMAL | OFF
```

| Mode | Safety | Performance | Use Case |
|------|--------|-------------|----------|
| `FULL` | Maximum | Slowest | Financial/critical data |
| `NORMAL` ✅ | Very High | Fast | Production web apps (recommended) |
| `OFF` ⚠️ | Low | Fastest | Development only (risky!) |

**⚠️ Warning**: `OFF` can cause data corruption on power loss!

---

## Verification

### Check Current Settings

```bash
# Connect to database
sqlite3 data/app.db

# Check all settings
PRAGMA journal_mode;           -- Should be: wal
PRAGMA synchronous;            -- Should be: 1 (NORMAL)
PRAGMA cache_size;             -- Should be: -<value>
PRAGMA mmap_size;              -- Should be: <value>
PRAGMA busy_timeout;           -- Should be: <value>
PRAGMA wal_autocheckpoint;     -- Should be: <value>
PRAGMA temp_store;             -- Should be: 2 (MEMORY)

# Exit
.exit
```

### Verify from Application Logs

When server starts, you'll see:
```
SQLite optimizations: journal_mode=WAL, synchronous=NORMAL, cache_size=32000KB, mmap_size=536870912KB, wal_autocheckpoint=2000, busy_timeout=7500ms
```

---

## Troubleshooting

### Database Locked Errors

**Symptoms**: `database is locked`, `PRAGMA busy_timeout` doesn't help

**Solutions**:
1. Increase busy_timeout: `PRAGMA busy_timeout = 15000`
2. Reduce MaxOpenConns: `db.SetMaxOpenConns(10)`
3. Check for long-running transactions
4. Enable WAL mode: `PRAGMA journal_mode = WAL`

---

### High Memory Usage

**Symptoms**: OOM killer, server swap thrashing

**Solutions**:
1. Reduce cache_size: `PRAGMA cache_size = -8000` (8MB)
2. Reduce MaxOpenConns: `db.SetMaxOpenConns(10)`
3. Reduce mmap_size: `PRAGMA mmap_size = 134217728` (128MB)

---

### Slow Queries

**Symptoms**: P99 latency > 100ms

**Solutions**:
1. Increase cache_size: `PRAGMA cache_size = -64000` (64MB)
2. Add indexes for frequent queries
3. Run ANALYZE: `sqlite3 data/app.db "ANALYZE;"`
4. Check query plans: `EXPLAIN QUERY PLAN <your-query>`

---

### Write Contention

**Symptoms**: Timeouts on write operations, WAL file grows large

**Solutions**:
1. Increase wal_autocheckpoint: `PRAGMA wal_autocheckpoint = 5000`
2. Run manual checkpoint: `PRAGMA wal_checkpoint(PASSIVE)`
3. Reduce concurrent writes (queue/batch writes)
4. **Consider PostgreSQL** if write-heavy

---

## Migration Guide

### From Development to Production

**Step 1**: Update configuration in `main.go`:

```go
// Change from development (512MB-1GB) to production (2-4GB)
db.SetMaxOpenConns(10)  // → 25
db.SetMaxIdleConns(3)   // → 10

db.Exec("PRAGMA cache_size = -8000")      // → -32000 (32MB)
db.Exec("PRAGMA mmap_size = 134217728")   // → 536870912 (512MB)
db.Exec("PRAGMA busy_timeout = 5000")     // → 7500 (7.5s)
db.Exec("PRAGMA wal_autocheckpoint = 1000") // → 2000
```

**Step 2**: Deploy and monitor:
```bash
# Deploy
git push && ssh user@server "cd laju-go && git pull && go build && sudo systemctl restart laju-go"

# Monitor logs
journalctl -u laju-go -f

# Check SQLite settings
sqlite3 data/app.db "PRAGMA cache_size;"
```

---

### From SQLite to PostgreSQL (Future Scaling)

**When to migrate**:
- Write-heavy workload (>30% writes)
- >10,000 concurrent users
- Need horizontal scaling
- Need high-availability (multi-region)

**Migration steps**:
1. Install PostgreSQL
2. Update connection string in `.env`
3. Replace SQLite-specific PRAGMAs with PostgreSQL equivalents
4. Run migrations
5. Test thoroughly

---

## Performance Benchmarks

### Vultr HF 1GB (1 vCPU, 1GB RAM, NVMe)

```
Configuration:
- MaxOpenConns: 10
- cache_size: 8MB
- mmap_size: 128MB

Benchmark (hey -n 1000 -c 10):
- Requests/sec: ~3,500
- P50 latency: 15ms
- P95 latency: 45ms
- P99 latency: 80ms
```

---

### Vultr HF 2GB (2 vCPU, 2GB RAM, NVMe)

```
Configuration:
- MaxOpenConns: 25
- cache_size: 32MB
- mmap_size: 512MB

Benchmark (hey -n 10000 -c 50):
- Requests/sec: ~15,000
- P50 latency: 8ms
- P95 latency: 25ms
- P99 latency: 45ms
```

---

### Vultr HF 4GB (4 vCPU, 4GB RAM, NVMe)

```
Configuration:
- MaxOpenConns: 50
- cache_size: 64MB
- mmap_size: 1GB

Benchmark (hey -n 50000 -c 100):
- Requests/sec: ~35,000
- P50 latency: 5ms
- P95 latency: 15ms
- P99 latency: 25ms
```

---

### Vultr HF 8GB (6 vCPU, 8GB RAM, NVMe)

```
Configuration:
- MaxOpenConns: 75
- cache_size: 256MB
- mmap_size: 1GB

Benchmark (hey -n 100000 -c 200):
- Requests/sec: ~60,000
- P50 latency: 3ms
- P95 latency: 10ms
- P99 latency: 18ms
```

---

### Vultr HF 16GB/58GB (16 vCPU, 16-58GB RAM, NVMe)

```
Configuration:
- MaxOpenConns: 100
- cache_size: 4GB
- mmap_size: 2GB

Expected (hey -n 200000 -c 500):
- Requests/sec: ~100,000-120,000
- P50 latency: 2ms
- P95 latency: 8ms
- P99 latency: 12ms
```

---

## Best Practices

### 1. Start Conservative, Scale Up

```go
// Start with minimal configuration
db.SetMaxOpenConns(10)
db.Exec("PRAGMA cache_size = -8000")

// Monitor usage for 1-2 weeks
// Increase if needed
```

### 2. Monitor Memory Usage

```bash
# Check memory usage
ps aux | grep laju-go

# Check SQLite memory
sqlite3 data/app.db "PRAGMA cache_size;"
```

### 3. Test Before Production

```bash
# Load test with new configuration
hey -n 10000 -c 50 http://localhost:8080/

# Monitor response times
# Check for errors
```

### 4. Document Changes

Always document configuration changes:
- Date of change
- Reason for change
- Before/after values
- Performance impact

### 5. Keep WAL Files Managed

```bash
# Check WAL file size
ls -lh data/app.db*

# Manual checkpoint if WAL is too large
sqlite3 data/app.db "PRAGMA wal_checkpoint(PASSIVE);"
```

---

## Quick Reference Table

| Server RAM | MaxOpenConns | cache_size | mmap_size | busy_timeout | wal_autocheckpoint |
|------------|--------------|------------|-----------|--------------|-------------------|
| 512MB ⚠️ | 10 | 8MB | 128MB | 5000 | 1000 |
| 1GB | 15 | 16MB | 256MB | 5000 | 1000 |
| 2GB ✅ | 25 | 32MB | 512MB | 7500 | 2000 |
| 4GB | 50 | 64MB | 1GB | 10000 | 3000 |
| 8GB | 75 | 256MB | 1GB | 10000 | 3000 |
| 16GB | 100 | 500MB | 2GB | 10000 | 5000 |
| 32GB | 100 | 2GB | 2GB | 10000 | 5000 |
| 58GB 🚀 | 100 | 4GB | 2GB | 10000 | 5000 |

---

## Related Documentation

- [Database Guide](../guide/database.md) - Database setup and migrations
- [Performance Optimization](optimization.md) - Query optimization and indexing
- [Production Deployment](production.md) - Production setup guide
- [Architecture Guide](../guide/architecture.md) - Repository pattern

---

## Changelog

| Date | Change | Reason |
|------|--------|--------|
| 2026-03-28 | Added 58GB enterprise config | Vultr HF 16vCPU/58GB optimization |
| 2026-03-28 | Updated default to 1-2GB config | Better balance for most users |
| 2026-03-28 | Added workload-specific configs | Read-heavy vs write-heavy tuning |
