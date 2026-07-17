# SQLite Data Protection & Recovery Guide

Complete guide to protecting production data from loss and implementing effective recovery strategies.

## Table of Contents

1. [Understanding Database Locks](#understanding-database-locks)
2. [Data Loss Scenarios](#data-loss-scenarios)
3. [Protection Strategies](#protection-strategies)
4. [Backup Implementation](#backup-implementation)
5. [Recovery Procedures](#recovery-procedures)
6. [Monitoring & Alerts](#monitoring--alerts)
7. [Production Checklist](#production-checklist)

---

## Understanding Database Locks

### Is Locked = Data Loss?

**Short answer: NO**

- Database locked is a **temporary condition**
- Data remains safe in WAL file
- Once lock is released, data is automatically committed

**Data loss ONLY occurs if:**

1. ❌ Power loss before `fsync()` completes
2. ❌ Disk corruption
3. ❌ WAL file deleted manually
4. ❌ Catastrophic hardware failure

---

## Data Loss Scenarios

### Scenario 1: Database Locked (NOT Data Loss)

```
Situation:
- Application gets "database is locked" error
- Users can't write temporarily

Data Status:
✅ Data SAFE in WAL file
✅ No corruption
✅ No manual intervention needed

Recovery:
1. Wait for lock to release (automatic)
2. Retry transaction (automatic with retry logic)
3. If persistent: Check long-running queries

Data Loss: ❌ NONE
```

---

### Scenario 2: Power Loss During Write

```
Timeline:
T0: Transaction starts
T1: Write to WAL file
T2: ⚡ POWER FAILURE! (before fsync)
T3: Power restored

Data Status:
⚠️ Last transaction MAY be lost
✅ Previous transactions SAFE
✅ Database NOT corrupted

Recovery:
1. SQLite auto-recovers on startup
2. WAL checkpoint validates integrity
3. Committed transactions preserved
4. Uncommitted transaction rolled back

Data Loss: ⚠️ Only last ~1 second of writes
```

---

### Scenario 3: WAL File Corruption

```
Situation:
- WAL file corrupted (disk error, bug, etc.)
- Main database intact

Data Status:
✅ Main database SAFE
⚠️ Recent writes in WAL may be lost

Recovery:
sqlite3 data/app.db "PRAGMA wal_checkpoint(TRUNCATE);"

// If that fails:
rm data/app.db-wal  // Delete corrupted WAL
sqlite3 data/app.db "PRAGMA wal_checkpoint(RESTART);"

Data Loss: ⚠️ Only uncommitted WAL transactions
```

---

### Scenario 4: Complete Database Corruption

```
Situation:
- Main database file corrupted
- WAL file may also be corrupted

Data Status:
❌ Database unreadable
❌ Cannot recover from WAL

Recovery:
1. Restore from backup (see backup strategy below)
2. Run integrity check
3. Migrate any data from WAL if possible

Data Loss: ❌ Depends on backup recency
```

---

## Protection Strategies

### Layer 1: SQLite Built-in Protection

```go
// 1. Enable WAL mode (already done)
db.Exec("PRAGMA journal_mode = WAL");

// 2. Set synchronous mode
db.Exec("PRAGMA synchronous = NORMAL");  // Balanced
// OR
db.Exec("PRAGMA synchronous = FULL");    // Maximum safety

// 3. Enable foreign keys
db.Exec("PRAGMA foreign_keys = ON");

// 4. Set busy timeout
db.Exec("PRAGMA busy_timeout = 5000");

// 5. Regular integrity checks
func checkIntegrity(db *sql.DB) error {
    var result string
    err := db.QueryRow("PRAGMA integrity_check").Scan(&result)
    if err != nil || result != "ok" {
        return fmt.Errorf("integrity check failed: %v", result)
    }
    return nil
}
```

---

### Layer 2: Automated Backups

#### Option A: SQLite Online Backup via CLI (Recommended)

Use the `sqlite3` CLI for consistent online backups. This works with any Go SQLite driver.

```bash
#!/bin/bash
# scripts/backup.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/opt/laju-go/backups"
DB_PATH="/opt/laju-go/data/app.db"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Online backup (no downtime)
sqlite3 "$DB_PATH" ".backup '$BACKUP_DIR/app-$DATE.db'"

# Delete backups older than 30 days
find "$BACKUP_DIR" -name "app-*.db" -mtime +30 -delete

echo "Backup completed: $DATE"
```

**Schedule with cron**:

```bash
# Daily backup at 2 AM
0 2 * * * /opt/laju-go/scripts/backup.sh
```

<!-- BackupService tidak ada di actual codebase. Backup via cron + sqlite3 CLI saja. -->
```

---

#### Option B: Simple File Copy (Easier)

```bash
#!/bin/bash
# scripts/backup.sh

set -e

BACKUP_DIR="./backups"
DB_PATH="./data/app.db"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Copy database files (WAL mode creates 3 files)
cp "$DB_PATH" "$BACKUP_DIR/backup_${TIMESTAMP}.db"
cp "$DB_PATH-shm" "$BACKUP_DIR/backup_${TIMESTAMP}.db-shm" 2>/dev/null || true
cp "$DB_PATH-wal" "$BACKUP_DIR/backup_${TIMESTAMP}.db-wal" 2>/dev/null || true

# Compress backup
cd "$BACKUP_DIR"
tar -czf "backup_${TIMESTAMP}.tar.gz" backup_${TIMESTAMP}.*
rm backup_${TIMESTAMP}.db*

# Cleanup old backups (keep last 7 days)
find "$BACKUP_DIR" -name "backup_*.tar.gz" -mtime +7 -delete

echo "Backup completed: backup_${TIMESTAMP}.tar.gz"
```

**Cron job (every 6 hours):**

```bash
# crontab -e
0 */6 * * * cd /path/to/laju-go && ./scripts/backup.sh >> /var/log/laju-backup.log 2>&1
```

---

### Layer 3: WAL Checkpoint Strategy

```go
// Periodic checkpoint to move WAL data to main database
func autoCheckpoint(db *sql.DB) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        // PASSIVE mode (non-blocking)
        var walSize, checkpointCount int
        err := db.QueryRow("PRAGMA wal_checkpoint(PASSIVE)").Scan(&walSize, &checkpointCount)
        if err != nil {
            log.Printf("Checkpoint error: %v", err)
            continue
        }

        // Log if checkpoint didn't complete
        if walSize > 0 {
            log.Printf("WAL checkpoint: %d pages remaining", walSize)
        }
    }
}
```

**Why important:**

- Moves data from WAL → main database
- Reduces WAL file size
- Faster recovery on startup
- Smaller backup files

---

### Layer 4: Replication (Advanced)

#### Option A: SQLite Replication with Litestream

```bash
# Install Litestream (real-time SQLite replication)
brew install litestream
# OR
go install github.com/benbjohnson/litestream/cmd/litestream@latest
```

**Configuration (`litestream.yml`):**

```yaml
dbs:
  - path: ./data/app.db
    replicas:
      - type: s3
        bucket: laju-go-backups
        path: prod/app.db
        access-key-id: ${AWS_ACCESS_KEY_ID}
        secret-access-key: ${AWS_SECRET_ACCESS_KEY}
        region: us-east-1
        retention: 168h  # Keep 7 days
        sync-interval: 1s  # Real-time replication
```

**Run Litestream:**

```bash
litestream replicate -config litestream.yml
```

**Benefits:**

- ✅ Real-time replication to S3
- ✅ Point-in-time recovery
- ✅ Automatic backup management
- ✅ Cross-region redundancy

---

#### Option B: rsync to Remote Server

```bash
#!/bin/bash
# scripts/sync-replica.sh

REMOTE_HOST="backup-server.example.com"
REMOTE_PATH="/backups/laju-go/"
LOCAL_DB="./data/app.db"

# Sync database files
rsync -avz \
    --delete \
    -e ssh \
    ./data/ \
    user@$REMOTE_HOST:$REMOTE_PATH

echo "Sync completed to $REMOTE_HOST"
```

---

### Layer 5: Monitoring & Alerts

```go
// app/services/health.go
package services

import (
    "database/sql"
    "fmt"
    "syscall"
    "time"
)

type HealthService struct {
    db *sql.DB
}

func (s *HealthService) CheckDatabase() error {
    // 1. Check connection
    if err := s.db.Ping(); err != nil {
        return fmt.Errorf("database connection failed: %v", err)
    }

    // 2. Check integrity
    var integrity string
    err := s.db.QueryRow("PRAGMA integrity_check").Scan(&integrity)
    if err != nil || integrity != "ok" {
        return fmt.Errorf("database integrity check failed: %s", integrity)
    }

    // 3. Check WAL size
    var walSize int
    err = s.db.QueryRow("PRAGMA wal_size").Scan(&walSize)
    if err != nil {
        return fmt.Errorf("cannot check WAL size: %v", err)
    }

    if walSize > 100_000_000 { // 100MB
        return fmt.Errorf("WAL file too large: %d bytes", walSize)
    }

    // 4. Check disk space
    stat := &syscall.Statfs_t{}
    err = syscall.Statfs("./data", stat)
    if err != nil {
        return fmt.Errorf("cannot check disk space: %v", err)
    }

    available := stat.Bavail * uint64(stat.Bsize)
    if available < 100_000_000 { // Less than 100MB
        return fmt.Errorf("low disk space: %d bytes available", available)
    }

    return nil
}

// StartMonitoring starts health check loop
func (s *HealthService) StartMonitoring(interval time.Duration, alertFunc func(error)) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        for range ticker.C {
            if err := s.CheckDatabase(); err != nil {
                alertFunc(err) // Send to Slack, email, etc.
            }
        }
    }()
}
```

---

## Recovery Procedures

### Recovery 1: After Lock Timeout

```go
func safeExecute(db *sql.DB, query string, args ...interface{}) error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        _, err := db.Exec(query, args...)
        if err == nil {
            return nil
        }

        if strings.Contains(err.Error(), "database is locked") {
            if i < maxRetries-1 {
                time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
                continue
            }
        }

        return err
    }
    return nil
}
```

---

### Recovery 2: After Power Failure

```bash
# 1. Check database integrity
sqlite3 data/app.db "PRAGMA integrity_check;"
# Expected: ok

# 2. Check WAL status
sqlite3 data/app.db "PRAGMA wal_checkpoint(PASSIVE);"
# Output: 0 0 (checkpointed, no remaining pages)

# 3. If WAL corrupted, delete and restart
rm data/app.db-wal
sqlite3 data/app.db "PRAGMA wal_checkpoint(RESTART);"

# 4. Restore from backup if needed
cp backups/backup_20260328_120000.db data/app.db
```

---

### Recovery 3: Complete Database Restore

```bash
#!/bin/bash
# scripts/restore.sh

BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: ./restore.sh <backup-file.tar.gz>"
    exit 1
fi

# Stop application
sudo systemctl stop laju-go

# Extract backup
tar -xzf "$BACKUP_FILE" -C ./data/

# Verify integrity
sqlite3 data/app.db "PRAGMA integrity_check;"

# Start application
sudo systemctl start laju-go

echo "Restore completed from $BACKUP_FILE"
```

---

## Production Checklist

### Prevention

- [ ] WAL mode enabled ✅
- [ ] `busy_timeout = 5000` or higher ✅
- [ ] `synchronous = NORMAL` (or FULL for critical data)
- [ ] Automated backups every 6 hours
- [ ] Backup retention: 7-30 days
- [ ] Off-site replication (S3 or remote server)
- [ ] WAL checkpoint monitoring
- [ ] Disk space monitoring
- [ ] Integrity check scheduled (weekly)

---

### Recovery

- [ ] Documented restore procedure
- [ ] Tested backup restoration
- [ ] Rollback plan for migrations
- [ ] Emergency contact list
- [ ] Runbook for common issues

---

### Monitoring

- [ ] Database health checks (every 5 min)
- [ ] WAL size alerts (>100MB)
- [ ] Backup success/failure alerts
- [ ] Disk space alerts (<1GB)
- [ ] Lock timeout tracking
- [ ] Error rate monitoring

---

## Data Loss Risk Assessment

| Scenario | Probability | Impact | Mitigation |
|----------|-------------|--------|------------|
| **Database locked** | High | Low (temporary) | Retry logic, busy_timeout |
| **Power loss** | Low | Medium (last tx) | synchronous=FULL, UPS |
| **WAL corruption** | Very Low | Low (recent tx) | Auto-checkpoint, backups |
| **Disk failure** | Low | High (all data) | Backups, replication |
| **Human error** | Medium | High | Backups, access control |

---

## Summary

**For production with critical data:**

1. **WAL mode** - Already enabled ✅
2. **Automated backups** - Every 6 hours, keep 7-30 days
3. **Off-site replication** - S3 or remote server
4. **Monitoring** - Health checks, alerts
5. **Tested recovery** - Practice restore procedures

**Expected data loss:**

- **With WAL + backups:** < 1 hour of data (usually seconds)
- **With Litestream:** < 1 second of data
- **Without WAL:** Minutes to hours

**Recovery time:**

- **Lock timeout:** Automatic (seconds)
- **Power failure:** Automatic (seconds)
- **Backup restore:** 5-30 minutes
- **Full disaster:** 1-4 hours

---

## Related Documentation

- [SQLite Configuration Guide](deployment/sqlite-configuration.md) - Complete configuration reference
- [Database Guide](guide/database.md) - Database setup and migrations
- [Production Deployment](deployment/production.md) - Production setup guide
- [Performance Optimization](deployment/optimization.md) - Query optimization

---

## Changelog

| Date | Change | Reason |
|------|--------|--------|
| 2026-03-28 | Initial documentation | Complete data protection guide |
