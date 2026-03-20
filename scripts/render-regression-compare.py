#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import re
from pathlib import Path

def read_text(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8", errors="ignore")

def find_first(root: Path, pattern: str):
    matches = sorted(root.glob(pattern))
    return matches[0] if matches else None

def parse_soak_report(path: Path) -> dict[str, str]:
    text = read_text(path)
    out: dict[str, str] = {}
    patterns = {
        "duration": r"- duration: (.+)",
        "errors": r"- errors: (.+)",
        "watch_events": r"- watch events: (.+)",
        "goroutines_before": r"- goroutines before: (.+)",
        "goroutines_after": r"- goroutines after: (.+)",
        "control_heartbeat": r"- control heartbeat count/errors/max: (.+)",
        "fault_counters": r"- fault log counters (.+)",
    }
    for key, pattern in patterns.items():
        m = re.search(pattern, text)
        if m:
            out[key] = m.group(1).strip()
    return out

def parse_csv_last(path: Path) -> dict[str, str]:
    if not path.exists():
        return {}
    with path.open("r", encoding="utf-8", newline="") as f:
        rows = list(csv.DictReader(f))
    return rows[-1] if rows else {}

def extract_iter43_baseline(doc: Path) -> str:
    text = read_text(doc)
    m = re.search(r"### mixed workload 延迟形态(.*?)### metadata benchmark 状态", text, re.S)
    if m:
        return m.group(1).strip()
    return "见 docs/architecture/test-report-iter43-stress-suite.md"

def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True)
    ap.add_argument("--output", required=True)
    ap.add_argument("--repo-root", default=".")
    args = ap.parse_args()

    root = Path(args.root)
    output = Path(args.output)
    repo_root = Path(args.repo_root)

    stress_dir = root / "stress"
    soak_dir = root / "soak"
    stress_report = stress_dir / "stress-test-report.md"
    integration_log = stress_dir / "integration-stress-run.log"
    mixed_log = find_first(stress_dir, "realistic-pressure-*.log")
    bench_log = stress_dir / "metadata-bench-run.log"
    soak_report = soak_dir / "sampled-soak-report.md"
    soak_csv = soak_dir / "sampled-soak-samples.csv"
    soak_run_log = soak_dir / "sampled-soak-run.log"

    soak_fields = parse_soak_report(soak_report)
    soak_last = parse_csv_last(soak_csv)
    baseline = extract_iter43_baseline(repo_root / "docs/architecture/test-report-iter43-stress-suite.md")

    lines = []
    lines.append("# Iter 48 regression compare report")
    lines.append("")
    lines.append("## Output roots")
    lines.append("")
    lines.append(f"- root: `{root}`")
    lines.append(f"- stress: `{stress_dir}`")
    lines.append(f"- soak: `{soak_dir}`")
    lines.append("")
    lines.append("## Artifact status")
    lines.append("")
    for label, path in [
        ("stress report", stress_report),
        ("integration log", integration_log),
        ("mixed log", mixed_log),
        ("benchmark log", bench_log),
        ("soak report", soak_report),
        ("soak csv", soak_csv),
        ("soak run log", soak_run_log),
    ]:
        if path is None:
            lines.append(f"- {label}: missing")
        else:
            lines.append(f"- {label}: {'present' if path.exists() else 'missing'} (`{path}`)")
    lines.append("")
    lines.append("## Iter 43 baseline reference")
    lines.append("")
    lines.append(baseline)
    lines.append("")
    lines.append("## Current soak summary")
    lines.append("")
    if soak_fields:
        for key, value in soak_fields.items():
            lines.append(f"- {key}: {value}")
    else:
        lines.append("- soak report not present or not yet generated")
    lines.append("")
    lines.append("## Current soak last-sample snapshot")
    lines.append("")
    if soak_last:
        for key in ["at", "goroutines", "heap_alloc", "heap_objects", "sessions_total", "sessions_active", "handles", "watch_events", "total_backlog", "control_resume_count", "control_heartbeat_count", "fault_logged"]:
            if key in soak_last:
                lines.append(f"- {key}: {soak_last[key]}")
    else:
        lines.append("- soak csv not present or empty")
    lines.append("")
    lines.append("## Notes")
    lines.append("")
    lines.append("- This report unifies the existing stress and soak artifacts into one markdown output.")
    lines.append("- For exact latency details, inspect the raw stress logs and the soak report together.")
    lines.append("- The main purpose is to compare a fresh local Iter 48 run against the Iter 43 baseline.")
    lines.append("")

    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("\n".join(lines), encoding="utf-8")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
