#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist/stress}"
DURATION="${DURATION:-3m}"
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-1s}"
BROWSE_WORKERS="${BROWSE_WORKERS:-4}"
SAVE_WORKERS="${SAVE_WORKERS:-3}"
HEARTBEAT_WORKERS="${HEARTBEAT_WORKERS:-2}"
RESUME_WORKERS="${RESUME_WORKERS:-2}"
FAULT_SLOW_CLIENT="${FAULT_SLOW_CLIENT:-1}"
FAULT_HALF_CLOSE="${FAULT_HALF_CLOSE:-1}"
FAULT_DELAYED_WRITE="${FAULT_DELAYED_WRITE:-1}"
EXTRA_ARGS="${EXTRA_ARGS:-}"

mkdir -p "$OUT_DIR"
cd "$ROOT"

go run ./cmd/devmount-soak   -duration "$DURATION"   -sample-interval "$SAMPLE_INTERVAL"   -browse-workers "$BROWSE_WORKERS"   -save-workers "$SAVE_WORKERS"   -heartbeat-workers "$HEARTBEAT_WORKERS"   -resume-workers "$RESUME_WORKERS"   -fault-slow-client=$([[ "$FAULT_SLOW_CLIENT" == "1" ]] && echo true || echo false)   -fault-half-close=$([[ "$FAULT_HALF_CLOSE" == "1" ]] && echo true || echo false)   -fault-delayed-write=$([[ "$FAULT_DELAYED_WRITE" == "1" ]] && echo true || echo false)   -csv "$OUT_DIR/sampled-soak-samples.csv"   -report "$OUT_DIR/sampled-soak-report.md"   $EXTRA_ARGS
