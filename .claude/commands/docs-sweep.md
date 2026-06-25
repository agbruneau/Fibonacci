---
description: Docs Sweep — sync one drifted derived doc against the code, verify, repeat until clean
---

# Docs Sweep Loop (FibGo)

Goal: keep **derived documentation** aligned with the code. One bounded slice per
iteration. Stop when a full pass finds zero drift.

## Each iteration

1. **Detect drift** — compare what the docs claim against ground truth:
   - Counts (packages / LOC): truth = `make stats`. Flag any doc that hardcodes a
     stale number (`README.md`, `docs/architecture/*`). CLAUDE.md says these must
     never be hardcoded-and-dated.
   - Package table / layering in `README.md` (Architecture section) vs `go list ./...`.
   - `docs/architecture/dependency-graph.mermaid` + C4 diagrams
     (`system-context`, `container-diagram`, `component-diagram`, `flows/*`) vs the
     real import graph. A package/edge that no longer exists, or a new one missing.
   - `docs/architecture/validation/validation-report.md` if it references moved/renamed symbols.

2. **Pick ONE drift** (highest signal: wrong counts > wrong package list > stale diagram).

3. **Fix it surgically.** Docs only. Smallest diff that makes the doc true.

4. **Verify before claiming done:**
   - Re-run `make stats`; the corrected number must match.
   - For graph/table fixes: re-derive from `go list ./...` and confirm the doc now matches.
   - Confirm the diff touches **only** `.md` / `.mermaid` files (`git diff --name-only`).
   - Sanity gate still green: `pwsh scripts/check.ps1` (build/vet/test/coverage are
     docs-agnostic, but `TestArchitectureLayering` guards the layering you may have re-described).

5. **Report**: what was stale, the truth, the one-line fix. Leave the change reviewable
   (working tree or a `docs:` commit on a branch — do not push without asking).

## NEVER touch

- `internal/**` or any `*.go` (this is a docs loop, not a refactor).
- `fibonacci/testdata/fibonacci_golden.json` (immutable, ADR-gated).
- `docs/dashboard/**` (generated artifact — regen via pnpm, never hand-edit).
- Anything an ADR governs — flag it, don't rewrite it.

## Stop condition

A full detection pass (step 1) that surfaces no drift → report "docs in sync, nothing to do"
and end the loop. Do not invent work to stay busy.
