#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/../developer-mount-docfix-release.zip}"
cd "$ROOT"
rm -f "$OUT"
zip -r "$OUT" README.md Task.md go.mod .gitignore cmd internal docs scripts configs dist >/dev/null
echo "Packaged release to $OUT"
