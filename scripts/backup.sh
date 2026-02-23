#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="${SCRIPT_DIR}/../backups"
DATA_DIR="${SCRIPT_DIR}/../data/postgres_data"

CONTAINER_PROD="job-tracker-db"
CONTAINER_DEV="job-tracker-db-dev"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Detect which PostgreSQL container is running
detect_container() {
    local container=""
    
    # Check both prod and dev containers
    if podman ps --format "{{.Names}}" 2>/dev/null | grep -q "^${CONTAINER_PROD}$"; then
        container="$CONTAINER_PROD"
    elif podman ps --format "{{.Names}}" 2>/dev/null | grep -q "^${CONTAINER_DEV}$"; then
        container="$CONTAINER_DEV"
    fi
    
    echo "$container"
}

# Stop PostgreSQL container
stop_postgres() {
    local container="$1"
    if podman ps --format "{{.Names}}" 2>/dev/null | grep -q "$container"; then
        log_info "Stopping PostgreSQL container ($container)..."
        podman stop "$container" >/dev/null
    else
        log_error "PostgreSQL container $container is not running"
        exit 1
    fi
}

# Start PostgreSQL container
start_postgres() {
    local container="$1"
    if podman ps -a --format "{{.Names}}" 2>/dev/null | grep -q "$container"; then
        log_info "Starting PostgreSQL container ($container)..."
        podman start "$container"
    fi
}

# Create backup
cmd_backup() {
    local container
    container=$(detect_container)
    
    if [[ -z "$container" ]]; then
        log_error "No PostgreSQL container running. Start either dev or prod first."
        exit 1
    fi
    
    # Create backup directory
    local timestamp
    timestamp=$(date +%Y%m%d%H%M%S)
    local backup_path="${BACKUP_DIR}/postgres_${timestamp}"
    
    mkdir -p "$BACKUP_DIR"
    
    log_info "Backing up PostgreSQL data..."
    log_info "Using container: $container"
    
    # Stop PostgreSQL
    stop_postgres "$container"
    
    # Copy data
    cp -r "$DATA_DIR" "$backup_path"
    
    # Start PostgreSQL
    start_postgres "$container"
    
    log_info "Backup created: $backup_path"
}

# List backups
cmd_list() {
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_info "No backups found"
        return
    fi
    
    local count=0
    echo ""
    echo "Available backups:"
    echo "------------------"
    for dir in "$BACKUP_DIR"/postgres_*/; do
        if [[ -d "$dir" ]]; then
            local name
            name=$(basename "$dir")
            local size
            size=$(du -sh "$dir" 2>/dev/null | cut -f1)
            echo "  $name ($size)"
            count=$((count + 1))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        echo "  No backups found"
    fi
    echo ""
}

# Restore backup
cmd_restore() {
    local container
    container=$(detect_container)
    
    if [[ -z "$container" ]]; then
        log_error "No PostgreSQL container running. Start either dev or prod first."
        exit 1
    fi
    
    # List available backups
    local backup_count=0
    local backups=()
    
    for dir in "$BACKUP_DIR"/postgres_*/; do
        if [[ -d "$dir" ]]; then
            backups+=("$(basename "$dir")")
            backup_count=$((backup_count + 1))
        fi
    done
    
    if [[ $backup_count -eq 0 ]]; then
        log_error "No backups found. Run './scripts/backup.sh backup' first."
        exit 1
    fi
    
    echo ""
    echo "Available backups:"
    echo "------------------"
    for i in "${!backups[@]}"; do
        local size
        size=$(du -sh "${BACKUP_DIR}/${backups[$i]}" 2>/dev/null | cut -f1)
        echo "  $((i + 1))) ${backups[$i]} ($size)"
    done
    echo ""
    
    # Prompt for selection
    read -p "Select backup to restore (1-${backup_count}): " selection
    
    if [[ ! "$selection" =~ ^[0-9]+$ ]] || [[ "$selection" -lt 1 ]] || [[ "$selection" -gt "$backup_count" ]]; then
        log_error "Invalid selection"
        exit 1
    fi
    
    local selected_backup="${backups[$((selection - 1))]}"
    
    log_warn "This will replace all current data with the backup!"
    read -p "Are you sure? (yes/no): " confirm
    
    if [[ "$confirm" != "yes" ]]; then
        log_info "Restore cancelled"
        exit 0
    fi
    
    log_info "Restoring from: $selected_backup"
    log_info "Using container: $container"
    
    # Stop PostgreSQL
    stop_postgres "$container"
    
    # Remove existing data and copy backup
    rm -rf "$DATA_DIR"
    cp -r "${BACKUP_DIR}/${selected_backup}" "$DATA_DIR"
    
    # Start PostgreSQL
    start_postgres "$container"
    
    log_info "Restore complete!"
}

# Show usage
usage() {
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  backup          Create a backup (auto-detects running PostgreSQL)"
    echo "  list            List available backups"
    echo "  restore         Restore from a backup (auto-detects running PostgreSQL)"
    echo ""
    echo "Examples:"
    echo "  $0 backup"
    echo "  $0 list"
    echo "  $0 restore"
}

# Main
case "${1:-}" in
    backup)
        cmd_backup
        ;;
    list)
        cmd_list
        ;;
    restore)
        cmd_restore
        ;;
    *)
        usage
        exit 1
        ;;
esac
