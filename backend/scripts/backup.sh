#!/bin/bash
set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-jobuser}"
DB_NAME="${DB_NAME:-jobtracker}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Generate timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/jobtracker_$TIMESTAMP.dump"

echo "Backing up database: $DB_NAME"
echo "Backup file: $BACKUP_FILE"

# Run pg_dump
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -Fc -f "$BACKUP_FILE"

echo "Backup completed successfully!"

# Keep only last 7 backups
if [ -d "$BACKUP_DIR" ]; then
    ls -t "$BACKUP_DIR"/*.dump | tail -n +8 | xargs -r rm
    echo "Old backups cleaned up (keeping last 7)"
fi
