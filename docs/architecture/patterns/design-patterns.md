# Design Patterns — FibGo

> Interactive architecture map: **[agbruneau.github.io/FibGo/dashboard/](https://agbruneau.github.io/FibGo/dashboard/)** (knowledge graph, 957 nodes / 8 layers / 13-step tour)

This document enumerates the concrete design patterns in use across the
FibGo codebase and points to the implementation sites. It is a companion
to [`interface-hierarchy.mermaid`](./interface-hierarchy.mermaid) and
complements the ADRs indexed in [`docs/ARCH.md`](../../ARCH.md#14-architectural-decision-records-adr).

## Inventory

| Pattern          | Intent                                       | Implementation site                                                                 |
|------------------|----------------------------------------------|-------------------------------------------------------------------------------------|
| **Strategy**     | Pluggable Fibonacci algorithms behind a uniform `Calculator` interface | `internal/fibonacci/calculator.go`, `fastdoubling.go`, `matrix.go`, `fft.go`        |
| **Factory / Registry** | Centralized calculator construction, decoupled from callers     | `internal/fibonacci/registry.go`                                                    |
| **Observer**     | Streaming progress updates to CLI / TUI / metrics sinks | `internal/progress/` — `progress.ProgressUpdate` channel                            |
| **Object Pool**  | `big.Int` recycling to eliminate GC pressure on large N. State-bound: `CalculationState` owns its arena, both share one lifecycle. | `sync.Pool`, `AcquireStateForN`/`ReleaseStateWithResult` in `internal/fibonacci/fastdoubling.go` |
| **Bump Allocator** | O(1) linear arena for FFT intermediate buffers and Fast Doubling state. | `internal/bigfft/bump.go`, `internal/fibonacci/memory/arena.go`                     |
| **Decorator**    | Cross-cutting capabilities layered on a base calculator | `FibCalculator` wrapping `CoreCalculator` in `internal/fibonacci/calculator.go` |
| **Facade**       | `app.Application` hides CLI parsing, dispatch, and error-to-exit-code mapping | `internal/app/app.go`                                                               |
| **Template Method** | `DoublingStepExecutor` fixes the Fast Doubling skeleton while subclasses choose the multiplication backend | `internal/fibonacci/doubling_framework.go`                                          |
| **Cache (LRU)**  | Thread-safe plan cache for FFT twiddle tables | `internal/bigfft/fft_cache.go`                                                      |
| **Circuit Breaker** (light) | Memory-budget pre-flight exits before OOM | `internal/fibonacci/memory/budget.go`                                               |
| **Adapter**      | OS-specific memory probes exposed behind a stable interface | `internal/metrics/system/`                                                          |

## Cross-cutting concerns

- **Concurrency contract**: goroutines always have bounded lifecycle;
  error propagation uses `parallel.ErrorCollector` (see
  [`internal/parallel/doc.go`](../../../internal/parallel/doc.go)) or
  `errgroup` depending on the call site.
- **Resource ownership**: every `sync.Pool` Get is paired with a deferred
  Put in the same scope. Bump arenas must call `Reset` before reuse. When
  a pooled state owns an arena, every `*big.Int` slot must be detached
  (`s.FK = new(big.Int)` and friends) before the state is returned to the
  pool — otherwise the arena would alias data the next tenant overwrites
  (`clearStateAliases` in `internal/fibonacci/fastdoubling.go`).

## Notes

- This file documents *current* code; it is not an aspirational catalogue.
  If you add a new pattern, extend the table and cross-link the site.
- For inter-package relationships, see
  [`dependency-graph.mermaid`](../dependency-graph.mermaid).
