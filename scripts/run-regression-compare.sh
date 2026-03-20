#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT/dist/regression}"
STRESS_DIR="${STRESS_DIR:-$OUT_DIR/stress}"
SOAK_DIR="${SOAK_DIR:-$OUT_DIR/soak}"
REPORT="${REPORT:-$OUT_DIR/regression-compare-report.md}"
RUN_STRESS="${RUN_STRESS:-1}"
RUN_SOAK="${RUN_SOAK:-1}"
DRY_RUN="${DRY_RUN:-0}"

mkdir -p "$OUT_DIR" "$STRESS_DIR" "$SOAK_DIR"
cd "$ROOT"

if [[ "$DRY_RUN" == "1" ]]; then
  echo "ROOT=$ROOT"
  echo "OUT_DIR=$OUT_DIR"
  echo "STRESS_DIR=$STRESS_DIR"
  echo "SOAK_DIR=$SOAK_DIR"
  echo "REPORT=$REPORT"
  echo "RUN_STRESS=$RUN_STRESS"
  echo "RUN_SOAK=$RUN_SOAK"
  echo "stress command: OUT_DIR=$STRESS_DIR ./scripts/run-stress-suite.sh"
  echo "soak command: OUT_DIR=$SOAK_DIR ./scripts/run-sampled-soak.sh"
  echo "report command: python3 ./scripts/render-regression-compare.py --root $OUT_DIR --output $REPORT --repo-root $ROOT"
  exit 0
fi

if [[ "$RUN_STRESS" == "1" ]]; then
  OUT_DIR="$STRESS_DIR" ./scripts/run-stress-suite.sh
fi

if [[ "$RUN_SOAK" == "1" ]]; then
  OUT_DIR="$SOAK_DIR" ./scripts/run-sampled-soak.sh
fi

python3 ./scripts/render-regression-compare.py --root "$OUT_DIR" --output "$REPORT" --repo-root "$ROOT"
echo "Iter 48 regression compare complete. Report written to $REPORT"
