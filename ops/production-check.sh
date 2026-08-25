#!/usr/bin/env bash
set -euo pipefail

base_url="${NEW_API_URL:-http://127.0.0.1:3000}"

NEW_API_URL="$base_url" "$(dirname "$0")/healthcheck.sh"

if [[ "${RUN_BACKUP:-true}" == "true" ]]; then
  "$(dirname "$0")/backup-sqlite.sh"
fi

printf 'Production preflight passed for %s\n' "$base_url"
