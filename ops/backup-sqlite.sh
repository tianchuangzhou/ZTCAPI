#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
db_path="${SQLITE_PATH:-$repo_dir/one-api.db}"
backup_dir="${BACKUP_DIR:-$repo_dir/backups}"
keep_count="${BACKUP_KEEP_COUNT:-14}"

if [[ ! -f "$db_path" ]]; then
  printf 'SQLite database not found: %s\n' "$db_path" >&2
  exit 1
fi

if ! [[ "$keep_count" =~ ^[1-9][0-9]*$ ]]; then
  printf 'BACKUP_KEEP_COUNT must be a positive integer\n' >&2
  exit 1
fi

mkdir -p "$backup_dir"
timestamp="$(date +%Y%m%d-%H%M%S)"
backup_path="$backup_dir/one-api-$timestamp.db"

# SQLite's online backup command is consistent while the service is running.
sqlite3 "$db_path" ".timeout 5000" ".backup '$backup_path'"
sqlite3 "$backup_path" "PRAGMA quick_check;" | grep -qx 'ok'

shasum -a 256 "$backup_path" > "$backup_path.sha256"

backups=()
while IFS= read -r backup_item; do
  backups+=("$backup_item")
done < <(find "$backup_dir" -maxdepth 1 -type f -name 'one-api-*.db' -print | sort -r)
if (( ${#backups[@]} > keep_count )); then
  for old_backup in "${backups[@]:keep_count}"; do
    rm -f -- "$old_backup" "$old_backup.sha256"
  done
fi

printf 'Backup created: %s\n' "$backup_path"
printf 'Checksum: %s\n' "$backup_path.sha256"
