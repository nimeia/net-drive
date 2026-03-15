#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/../developer-mount.zip}"
cd "$ROOT"
rm -f "$OUT"
zip -r "$OUT" .
