#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_FILE="${1:-$ROOT_DIR/dist/benchmark-gate.out}"
mkdir -p "$(dirname "$OUT_FILE")"

cd "$ROOT_DIR"
go test ./internal/server ./internal/transport -bench . -benchmem -run '^$' | tee "$OUT_FILE"
go run ./cmd/devmount-benchgate -input "$OUT_FILE" -thresholds "$ROOT_DIR/benchmarks/thresholds.json"
