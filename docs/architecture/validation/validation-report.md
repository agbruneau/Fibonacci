# Architecture Validation

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour)

**Project**: FibCalc (github.com/agbruneau/FibGo)

This document records the architectural invariants that the documentation
suite asserts and that have been checked against the source code. It is a
living reference, not a point-in-time process artifact: when the structure
changes, re-verify the items below and update them in place.

---

## Layer tightness & dependency direction

The project follows Clean Architecture with the dependency hierarchy:

```
cmd → app → orchestration → fibonacci / bigfft → config / errors
```

Verified properties (against Go `import` declarations in source):

| Invariant | Status | Evidence |
|-----------|--------|----------|
| `orchestration` does not import `cli` | Holds | No `internal/cli` import in `internal/orchestration/*.go` |
| `orchestration` does not import `tui` | Holds | No `internal/tui` import in `internal/orchestration/*.go` |
| `cli` depends on `orchestration` (not the reverse) | Holds | `internal/cli` imports `internal/orchestration` |
| `tui` depends on `orchestration` (not the reverse) | Holds | `internal/tui` calls orchestration entry points |
| `app` is the composition root | Holds | `internal/app` imports `cli`, `tui`, `orchestration`, `fibonacci`, `config`, `errors` |
| `internal/**` does not leak into `cmd/**` | Holds | `cmd/fibcalc` delegates to `internal/app` only |

The four C4 / dependency Mermaid diagrams
(`system-context.mermaid`, `container-diagram.mermaid`,
`component-diagram.mermaid`, `dependency-graph.mermaid`) are consistent
with the import graph above: edges flow inward, with no
`orchestration → cli` / `orchestration → tui` back-edges, and
`cli → orchestration` / `tui → orchestration` present.

## Interface & pattern claims

The interface signatures and design-pattern catalogue in
[`../README.md`](../README.md) and
[`../patterns/design-patterns.md`](../patterns/design-patterns.md) are
maintained to match source. Notable narrow/wide interface contracts:

- `Calculator` (public) / `CoreCalculator` (internal) — `internal/fibonacci/calculator.go`
- `Multiplier` (narrow) extended by `DoublingStepExecutor` (wide)
- `ProgressObserver` — `internal/progress/` / `ProgressReporter` — `internal/orchestration/`
- `CalculatorFactory` — `internal/fibonacci/registry.go`

## Execution flows

Flow narratives live in [`docs/ARCH.md`](../../ARCH.md) (sections 6-8) and
the diagrams under [`../flows/`](../flows/):

- CLI path: `main()` → `app.New()` → `app.Run()` → orchestration → output
- TUI path: dispatch → `tui.Run()` → Bubble Tea model lifecycle
- Configuration resolution: CLI flags → env vars → calibration profile →
  adaptive estimation → defaults
- Per-algorithm: `FibCalculator` decorator → `DoublingFramework` /
  `MatrixFramework` → strategy dispatch

## Maintenance

When refactoring package boundaries, re-check the dependency-direction
table above and the Mermaid diagrams. A future tooling step (Makefile or
pre-commit) that parses documented interface signatures and compares
them against source (via reflection or AST) would let this drift be
detected automatically.
