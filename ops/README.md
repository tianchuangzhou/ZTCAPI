# Local Operations

## SQLite backup

Run a consistent backup while the service is online:

```bash
./ops/backup-sqlite.sh
```

The default destination is `./backups`, keeping the newest 14 database files. Override it with `BACKUP_DIR` and `BACKUP_KEEP_COUNT`.

Before a release, create a backup and verify its checksum:

```bash
BACKUP_DIR=/var/backups/new-api BACKUP_KEEP_COUNT=30 ./ops/backup-sqlite.sh
shasum -a 256 -c /var/backups/new-api/*.sha256
```

A production deployment should run this from a scheduler and periodically restore a copy into a separate test database.

Restore drill (stop the application first, then explicitly confirm):

```bash
RESTORE_CONFIRM=YES SQLITE_PATH=/var/lib/new-api/one-api.db \
  ./ops/restore-sqlite.sh /var/backups/new-api/one-api-20260824-010000.db
```

The script verifies `quick_check`, validates a sidecar SHA-256 file when present, and creates a pre-restore backup before replacing the live database.

## HTTPS reverse proxy

Use the included Caddy template after pointing your DNS record at the server:

```bash
cp ops/Caddyfile.example /etc/caddy/Caddyfile
# replace api.example.com and the upstream port, then: caddy validate --config /etc/caddy/Caddyfile
```

## Health check

Use the same command for a process supervisor, reverse proxy, or uptime monitor:

```bash
NEW_API_URL=http://127.0.0.1:3000 ./ops/healthcheck.sh
```

Run both the health check and a consistent SQLite backup before a release:

```bash
NEW_API_URL=http://127.0.0.1:3000 ./ops/production-check.sh
```
