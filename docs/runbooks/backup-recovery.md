# Backup and Recovery Runbook

## Overview

My Patreon Manager stores state in either SQLite (default) or PostgreSQL. This runbook covers backup and recovery procedures for both backends.

## SQLite Backup

### Prerequisites

- Access to the filesystem where the SQLite database file resides
- The database file path is configured via the `DB_PATH` environment variable (default: `patreon-manager.db` in the working directory)

### Backup Procedure

1. **Stop the application** (recommended for consistency, though SQLite supports hot backup via `.backup`):
   ```sh
   # If running as a service
   systemctl stop patreon-manager
   ```

2. **Copy the database file**:
   ```sh
   cp patreon-manager.db "patreon-manager-$(date +%Y%m%d-%H%M%S).db.bak"
   ```

3. **Verify the backup integrity**:
   ```sh
   sqlite3 "patreon-manager-$(date +%Y%m%d-%H%M%S).db.bak" "PRAGMA integrity_check;"
   ```
   Expected output: `ok`

4. **Restart the application**:
   ```sh
   systemctl start patreon-manager
   ```

### Hot Backup (without stopping)

If downtime is not acceptable, use SQLite's `.backup` command:

```sh
sqlite3 patreon-manager.db ".backup 'patreon-manager-backup.db'"
```

This creates a consistent snapshot even while the application is writing.

### Automated Backup Script

```sh
#!/bin/bash
BACKUP_DIR="/var/backups/patreon-manager"
DB_PATH="${DB_PATH:-patreon-manager.db}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"
sqlite3 "$DB_PATH" ".backup '$BACKUP_DIR/patreon-manager-$TIMESTAMP.db'"

# Retain last 30 days of backups
find "$BACKUP_DIR" -name "*.db" -mtime +30 -delete
```

### Recovery Procedure

1. Stop the application.
2. Replace the database file with the backup:
   ```sh
   cp patreon-manager-YYYYMMDD-HHMMSS.db.bak patreon-manager.db
   ```
3. Verify integrity:
   ```sh
   sqlite3 patreon-manager.db "PRAGMA integrity_check;"
   ```
4. Start the application.

---

## PostgreSQL Backup

### Prerequisites

- `pg_dump` and `psql` CLI tools installed
- Database connection details configured via environment variables (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`)

### Backup Procedure

1. **Create a compressed backup**:
   ```sh
   pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
     --format=custom --compress=9 \
     -f "patreon-manager-$(date +%Y%m%d-%H%M%S).dump"
   ```

2. **Verify the backup**:
   ```sh
   pg_restore --list "patreon-manager-YYYYMMDD-HHMMSS.dump" | head -20
   ```

### Plain SQL Backup (for portability)

```sh
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  --format=plain --no-owner --no-privileges \
  > "patreon-manager-$(date +%Y%m%d-%H%M%S).sql"
```

### Recovery Procedure

1. **Stop the application**.

2. **Restore from custom format dump**:
   ```sh
   pg_restore -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
     --clean --if-exists \
     "patreon-manager-YYYYMMDD-HHMMSS.dump"
   ```

3. **Or restore from plain SQL**:
   ```sh
   psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
     < "patreon-manager-YYYYMMDD-HHMMSS.sql"
   ```

4. **Start the application**.

### Automated PostgreSQL Backup Script

```sh
#!/bin/bash
BACKUP_DIR="/var/backups/patreon-manager"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  --format=custom --compress=9 \
  -f "$BACKUP_DIR/patreon-manager-$TIMESTAMP.dump"

# Retain last 30 days of backups
find "$BACKUP_DIR" -name "*.dump" -mtime +30 -delete
```

---

## Disaster Recovery Checklist

1. Identify the most recent valid backup.
2. Stop the application on all instances.
3. Restore the database using the appropriate procedure above.
4. Run `go run ./cmd/cli validate` to verify configuration.
5. Run `go run ./cmd/cli sync --dry-run` to verify the restored state.
6. Start the application.
7. Monitor logs and metrics for the next sync cycle to confirm normal operation.
