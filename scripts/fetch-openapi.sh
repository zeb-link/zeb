#!/usr/bin/env bash
set -euo pipefail

url="${1:-http://localhost:3000/api/v1/openapi.json}"
out="${2:-internal/openapi/openapi.json}"

mkdir -p "$(dirname "$out")"
curl -fsSL "$url" -o "$out"
printf 'Synced %s from %s\n' "$out" "$url"
