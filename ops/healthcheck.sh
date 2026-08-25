#!/usr/bin/env bash
set -euo pipefail

base_url="${NEW_API_URL:-http://127.0.0.1:3000}"
status_url="$base_url/api/status"

response="$(curl --fail --silent --show-error --max-time "${HEALTHCHECK_TIMEOUT:-5}" "$status_url")"
if ! printf '%s' "$response" | grep -q '"success"[[:space:]]*:[[:space:]]*true'; then
  printf 'New API returned an unhealthy response from %s\n' "$status_url" >&2
  exit 1
fi

printf 'New API healthy: %s\n' "$base_url"
