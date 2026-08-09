# Design Patterns — FibGo

> Interactive architecture map: **[agbruneau.github.io/Fibonacci/dashboard/](https://agbruneau.github.io/Fibonacci/dashboard/)** (knowledge graph, 1128 nodes / 4782 edges / 9 layers / 12-step tour — counts from the graph regenerated 2026-07-06; re-verify at the next regeneration)

This document enumerates the concrete design patterns in use across the
FibGo codebase and points to the implementation sites. It is a companion
to [`interface-hierarchy.mermaid`](./interface-hierarchy.mermaid) and
complements the canonical ADR registry in [`docs/adr/`](../../adr/).
(`docs/ARCH.md` §14 is a separate in-document narrative journal with its own
three-digit numbering — it does not index that registry; it links to it.)

## Inventory

| Pattern          | Intent                                       | Implementation site                                                                 |
|------------------|----------------------------------------------|-------------------------------------------------------------------------------------|
| **Strategy**     | Pluggable Fibonacci algorithms behind a uniform `Calculator` interface | `internal/fibonacci/calculator.go`, `fastdoubling.go`, `matrix.go`, `fft_based.go`  |
| **Factory / Registry** | Centralized calculator construction, decoupled from callers     | `internal/fibonacci/registry.go`                                                    |
| **Observer**     | Streaming progress updates to CLI / TUI / metrics sinks | `internal/progress/` — `progress.ProgressUpdate` channel                            |
| **Object Pool**  | `big.Int` recycling to eliminate GC pressure on large N. State-bound: `CalculationState` owns its arena, both share one lifecycle. | `sync.Pool`, `AcquireStateForN`/`ReleaseStateWithResult` in `internal/fibonacci/fastdoubling.go` |
| **Bump Allocator** | O(1) linear arena for FFT intermediate buffers and Fast Doubling state. | `internal/bigfft/bump.go`, `internal/fibonacci/memory/arena.go`                     |
| **Decorator**    | Cross-cutting capabilities layered on a base calculator | `FibCalculator` wrapping `CoreCalculator` in `internal/fibonacci/calculator.go` |
| **Facade**       | `app.Application` hides CLI parsing, dispatch, and error-to-exit-code mapping | `internal/app/app.go`                                                               |
| **Template Method** | `DoublingFramework` fixes the Fast Doubling skeleton while a pluggable `DoublingStepExecutor` supplies the multiplication backend | `internal/fibonacci/doubling_framework.go` (framework), `internal/fibonacci/strategy.go` (executor) |
| **Cache (LRU)**  | Thread-safe cache of forward-FFT transform results (`PolValues`) | `internal/bigfft/fft_cache.go`                                                      |
| **Circuit Breaker** (light) | Memory-budget pre-flight exits before OOM | `internal/fibonacci/memory/budget.go`                                               |
| **Adapter**      | Host CPU/memory probes (`gopsutil`) adapted into a bubbletea message | `internal/tui/commands.go` — `sampleSysStatsCmd` (inlined from the former `internal/metrics/system`) |

## Cross-cutting concerns

- **Concurrency contract**: goroutines always have bounded lifecycle;
  error propagation uses `errgroup` (or the allocation-free
  `parallel3Result` struct on the fastdoubling hot path — see
  `internal/fibonacci/common.go`).
- **Resource ownership**: every state acquisition is paired with a release
  in the same scope (`AcquireStateForN`/`ReleaseStateWithResult`, or the
  per-calculator `acquireStateForN`/`releaseStateWithResult`). Since commit
  fa13bfd the release sink is either the shared `sync.Pool` or a
  per-calculator GC-immune cache slot
  (`FastDoublingCalculator.cachedState`, bounded by `maxCachedArenaWords`)
  that retains the state across calls. Bump arenas must call `Reset`
  before reuse. When a pooled or cached state owns an arena, every
  `*big.Int` slot must be detached (`s.FK = new(big.Int)` and friends)
  before the state reaches either sink — otherwise the arena would alias
  data the next tenant overwrites (`clearStateAliases`, invoked
  unconditionally by `finalizeStateReleaseTo` in
  `internal/fibonacci/fastdoubling.go`).

## Notes

- This file documents *current* code; it is not an aspirational catalogue.
  If you add a new pattern, extend the table and cross-link the site.
- For inter-package relationships, see
  [`dependency-graph.mermaid`](../dependency-graph.mermaid).
