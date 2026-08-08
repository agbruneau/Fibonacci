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
cmd → app → orchestration → fibonacci → bigfft

config is a SIBLING of orchestration, not a layer below fibonacci: it is
imported by app, cli, tui and calibration, and by nothing inside
fibonacci or bigfft. The reverse arrow (config → fibonacci, config →
bigfft) is what internal/arch_test.go forbids; it tolerates the two
documented lateral imports config → fibonacci/memory and config → ui.

leaves (zero internal imports): bigfft, errors, format, metrics, progress,
                                ui, testutil, fibonacci/memory,
                                fibonacci/threshold, cli/completion
```

The `internal_test` package doc comment in `internal/arch_test.go` states the same chain.
Verified 2026-08-07 with
`go list -deps=false -f '{{.ImportPath}} => {{join .Imports " "}}' ./...`:
`internal/fibonacci` imports `bigfft`, `errors`, `fibonacci/memory`,
`fibonacci/threshold`, `progress` — no `config`; and `internal/bigfft`
imports no internal package at all.

Verified properties (against Go `import` declarations in source):

| Invariant | Status | Evidence |
|-----------|--------|----------|
| `orchestration` does not import `cli` | Holds | No `internal/cli` import in `internal/orchestration/*.go` |
| `orchestration` does not import `tui` | Holds | No `internal/tui` import in `internal/orchestration/*.go` |
| `cli` depends on `orchestration` (not the reverse) | Holds | `internal/cli` imports `internal/orchestration` |
| `tui` depends on `orchestration` (not the reverse) | Holds | `internal/tui` calls orchestration entry points |
| `app` is the composition root | Holds | `internal/app` imports `cli`, `tui`, `orchestration`, `fibonacci`, `config`, `errors` |
| `internal/**` does not leak into `cmd/**` | Holds | `cmd/fibcalc` delegates to `internal/app` only |

Only two of the diagrams carry package-import edges, and both were
re-verified arrow-by-arrow against `go list -f '{{range .Imports}}…'`
on 2026-08-07:

- `dependency-graph.mermaid` — exact and complete. Its 45 arrows match, one
  for one, the 45 direct internal imports `go list` reports across the module
  (`cmd/fibcalc` 1, `app` 10, `calibration` 7, `cli` 8, `config` 3,
  `fibonacci` 5, `orchestration` 4, `tui` 7; every other package is a leaf).
  (Previously re-verified 2026-07-11, audit Fable5 ARCH-01.)
- `container-diagram.mermaid` — every `Rel` **between two `Container`s** is a
  real import. The one exception is `Rel(user, entry, "Invokes")` (line 19),
  a `Person` → `Container` relation that is not an import at all. The
  `orch → config` edge the diagram used to draw was **false**:
  `internal/orchestration` imports only `errors`, `fibonacci`,
  `fibonacci/memory`, `progress`. It was removed on 2026-08-07, and the real
  `cli → config`, `tui → config`, `calib → config`, `calib → bigfft` edges plus
  the seven edges into the leaf-package container were added. On 2026-08-07 the
  leaf container was also widened from six packages to nine: it previously
  omitted `fibonacci/memory`, `fibonacci/threshold` and `cli/completion`, so
  `config → fibonacci/memory`, `app → fibonacci/memory`,
  `app → fibonacci/threshold`, `orch → fibonacci/memory`,
  `fibonacci → fibonacci/memory`, `fibonacci → fibonacci/threshold` and
  `cli → cli/completion` had no representation. With those nine leaves inside
  one container, the seven `Rel(*, support, …)` labels now enumerate every
  remaining direct internal import.

`system-context.mermaid` has no package edges at all — its four `Rel`s are
to a Person and three external systems. `component-diagram.mermaid` is a
`classDiagram`: its arrows are class relations, not imports, and they are
**not** covered by the import check above. Three of them were false and were
corrected on 2026-08-07:

- `CoreCalculator <|.. FibCalculator` was drawn as a realization.
  `*FibCalculator` has `Name`, `Calculate` and `CalculateWithObservers` and no
  `CalculateCore`, so it does not implement `CoreCalculator` — it composes one
  (`-core CoreCalculator`). Now an aggregation. Same fix applied in
  `patterns/interface-hierarchy.mermaid`.
- `DoublingFramework --> ProgressSubject` — neither framework ever receives a
  `*ProgressSubject`; `ExecuteDoublingLoop` and `ExecuteMatrixLoop` both take a
  `progress.ProgressCallback`. Retargeted.
- `AdaptiveStrategy --> TransformCache` — cached *transforms* are read only
  through `Mul`/`Sqr`/`MulTo`/`SqrTo` (`fftmulTo`/`fftsqrTo` →
  `MulCachedWithBump`/`SqrCachedWithBump`, called from
  `internal/bigfft/fft_core.go:fftmulTo` and `internal/bigfft/fft_core.go:fftsqrTo`),
  which the *matrix* path enters via `smartMultiply`/`smartSquare`.
  `executeDoublingStepFFT` calls `TransformWithBump`, not
  `TransformCachedWithBump`, and `AdaptiveStrategy.ExecuteStep` routes every
  operand above `FFTThreshold` there (`internal/fibonacci/strategy.go:AdaptiveStrategy.ExecuteStep`),
  so the operands that fall through to `smartMultiply` never clear its own FFT
  gate. Retargeted to `MatrixFramework`. Corrected further on 2026-08-07: the
  cache is nevertheless *configured* from two places outside that set —
  `configureFFTCache` → `SetTransformCacheConfig` once per calculation with
  `n > MaxFibUint64` (`internal/fibonacci/options.go:configureFFTCache`, called
  from `calculator.go:FibCalculator.CalculateWithObservers`), and `bigfftCacheStrategy.Sample` →
  `GetTransformCache`/`SetTransformCacheConfig` from *inside*
  `ExecuteDoublingLoop`, gated on a non-nil `DynamicThresholdManager` and
  throttled to every `cacheSampleInterval` iterations
  (`internal/fibonacci/doubling_framework.go:DoublingFramework.ExecuteDoublingLoop`,
  `cache_strategy_bigfft.go:bigfftCacheStrategy.Sample`). A
  `DoublingFramework --> TransformCache` edge was added for that path.

Class *members* (field/method listings in `component-diagram.mermaid`) drift
independently from both checks; they were last corrected on 2026-08-07 —
`tempAllocator`/`BumpAllocator`/`poolAllocator` methods and the
`DynamicThresholdManager` threshold getters had been shown as exported when the
source has them unexported, `AdaptiveStrategy` carried a field the `struct{}`
does not have, and `MatrixFramework` was drawn holding a `Multiplier` instead of
its `SquareFunc`. Re-verify members separately from the edge check.

## Interface & pattern claims

The interface signatures and design-pattern catalogue in
[`../README.md`](../README.md) and
[`../patterns/design-patterns.md`](../patterns/design-patterns.md) are
maintained to match source. Notable narrow/wide interface contracts:

- `Calculator` (decorated façade) / `CoreCalculator` (algorithm kernel wrapped by `FibCalculator`) — both exported, `internal/fibonacci/calculator.go`
- `Multiplier` (narrow) extended by `DoublingStepExecutor` (wide)
- `ProgressObserver` — `internal/progress/` / `ProgressReporter` — `internal/orchestration/`
- `CalculatorFactory` — `internal/fibonacci/registry.go`

## Execution flows

Flow narratives live in [`docs/ARCH.md`](../../ARCH.md) (sections 6-8) and
the diagrams under [`../flows/`](../flows/):

- CLI path: `main()` → `app.New()` → `app.Run()` → orchestration → output
- TUI path: dispatch → `tui.Run()` → Bubble Tea model lifecycle
- Configuration resolution: CLI flags → env vars → defaults for every setting,
  **except** `Threshold`/`FFTThreshold`/`StrassenThreshold`, which a valid cached
  calibration profile overwrites after `ParseConfig` has run
  (`internal/app/app.go:New`, `internal/calibration/calibration.go:LoadCachedCalibration`);
  adaptive estimation only fills thresholds still at 0 when no valid profile
  loads
- Per-algorithm: `FibCalculator` decorator → `DoublingFramework` /
  `MatrixFramework` → strategy dispatch

## Maintenance

When refactoring package boundaries, re-check the dependency-direction
table above and the Mermaid diagrams. A future tooling step (Makefile or
pre-commit) that parses documented interface signatures and compares
them against source (via reflection or AST) would let this drift be
detected automatically.
