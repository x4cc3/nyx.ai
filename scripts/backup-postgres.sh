#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "usage: $0 <database-url> <output-file>" >&2
  exit 1
fi

database_url="$1"
output_file="$2"

mkdir -p "$(dirname "$output_file")"
pg_dump "$database_url" --format=custom --file="$output_file"
echo "backup written to $output_file"
