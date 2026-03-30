#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "usage: $0 <database-url> <backup-file>" >&2
  exit 1
fi

database_url="$1"
backup_file="$2"

pg_restore --clean --if-exists --no-owner --dbname="$database_url" "$backup_file"
echo "restore completed from $backup_file"
