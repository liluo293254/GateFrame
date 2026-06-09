#!/bin/sh
set -e
DB_HOST="${DB_HOST:-ddev-gateframe-db}"
DB_PORT="${DB_PORT:-5432}"
echo "waiting for postgres at ${DB_HOST}:${DB_PORT}..."
for i in $(seq 1 60); do
  if nc -z "${DB_HOST}" "${DB_PORT}" 2>/dev/null; then
    echo "postgres is up"
    exec /usr/local/bin/file-service
  fi
  sleep 1
done
echo "postgres not ready after 60s" >&2
exit 1
