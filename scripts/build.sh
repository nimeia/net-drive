#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
mkdir -p "$DIST"
cd "$ROOT"
go build -o "$DIST/devmount-server" ./cmd/devmount-server
go build -o "$DIST/devmount-client" ./cmd/devmount-client
echo "Built binaries into $DIST"
