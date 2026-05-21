#!/usr/bin/env python3
"""bench_gate.py — fail CI if any benchstat ns/op delta exceeds the threshold.

Usage:
    bench_gate.py <benchstat-output-file> <max-pct-regression>

Reads a benchstat diff output (as produced by `benchstat -row /name old new`)
and exits non-zero when any benchmark row has a sec/op delta worse than
+max_pct%. A leading "+" indicates regression; "-" indicates improvement.

This is the regression-gate engine for Audit-PRD E4 (S2-T5).
"""
from __future__ import annotations

import re
import sys


def parse_threshold(arg: str) -> float:
    try:
        v = float(arg)
    except ValueError:
        raise SystemExit(f"bench_gate: invalid threshold {arg!r}; want a float (percent)")
    if v <= 0:
        raise SystemExit("bench_gate: threshold must be positive")
    return v


# benchstat -row /name emits a delta column of the form "  +12.3% (p=...)"
# or "~" when the change is below the noise floor. We capture the signed
# percentage if present.
DELTA_RE = re.compile(r"([+-]?\d+(?:\.\d+)?)%")


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: bench_gate.py <benchstat-output> <max-pct-regression>", file=sys.stderr)
        return 2

    diff_path = sys.argv[1]
    threshold = parse_threshold(sys.argv[2])

    try:
        with open(diff_path, "r", encoding="utf-8") as f:
            text = f.read()
    except OSError as exc:
        print(f"bench_gate: cannot read {diff_path}: {exc}", file=sys.stderr)
        return 2

    violations: list[tuple[str, float]] = []
    for line in text.splitlines():
        # Skip headers and blank lines.
        stripped = line.strip()
        if not stripped or stripped.startswith(("goos", "goarch", "pkg", "cpu", "─", "geomean")):
            continue
        match = DELTA_RE.search(line)
        if not match:
            continue
        try:
            delta = float(match.group(1))
        except ValueError:
            continue
        if delta > threshold:
            # First token of the line is the benchmark name (or row label).
            name = line.split()[0] if line.split() else "<unknown>"
            violations.append((name, delta))

    if violations:
        print("bench_gate: regression(s) exceeding threshold detected:", file=sys.stderr)
        for name, delta in violations:
            print(f"  {name}: +{delta:.2f}% (max allowed +{threshold:.2f}%)", file=sys.stderr)
        return 1

    print(f"bench_gate: no regression > {threshold:.2f}% detected.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
