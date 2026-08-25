#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
db_path="${SQLITE_PATH:-$repo_dir/one-api.db}"
backup_path="${1:-}"

if [[ -z "$backup_path" ]]; then
  printf 'Usage: RESTORE_CONFIRM=YES %s /path/to/one-api-YYYYMMDD-HHMMSS.db\n' "$0" >&2
  exit 2
fi
if [[ ! -f "$backup_path" ]]; then
  printf 'Backup not found: %s\n' "$backup_path" >&2
  exit 1
fi
if [[ "${RESTORE_CONFIRM:-}" != "YES" ]]; then
  printf 'Restore replaces %s. Set RESTORE_CONFIRM=YES after stopping the service.\n' "$db_path" >&2
  exit 2
fi

sqlite3 "$backup_path" "PRAGMA quick_check;" | grep -qx 'ok'
if [[ -f "$backup_path.sha256" ]]; then
  shasum -a 256 -c "$backup_path.sha256"
fi

mkdir -p "$(dirname "$db_path")"
if [[ -f "$db_path" ]]; then
  pre_restore="$db_path.pre-restore-$(date +%Y%m%d-%H%M%S)"
  sqlite3 "$db_path" ".timeout 5000" ".backup '$pre_restore'"
  printf 'Current database backed up to: %s\n' "$pre_restore"
fi
tmp_path="$db_path.restore.tmp"
rm -f -- "$tmp_path"
sqlite3 "$backup_path" ".timeout 5000" ".backup '$tmp_path'"
sqlite3 "$tmp_path" "PRAGMA quick_check;" | grep -qx 'ok'
mv -f -- "$tmp_path" "$db_path"
printf 'Restore completed: %s\n' "$db_path"
