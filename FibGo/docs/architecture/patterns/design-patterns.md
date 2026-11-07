# Design Patterns — FibGo

**Authoritative inventory.** It is here — and nowhere else — that the list
of patterns in use in FibGo is kept, each with its rationale and its implementation
site. [`docs/ARCH.md` §5](../../ARCH.md#5-design-patterns) points to it and keeps
no copy: until 2026-09-04 the two files each kept their own table
(14 entries on one side, 11 on the other, different sets, no mutual reference);
the table below is their union, reconciled against the source.

Companion: the [`interface-hierarchy.md`](./interface-hierarchy.md) figure for the
interfaces involved. Canonical decision register: [`docs/adr/`](../../adr/)
(`docs/ARCH.md` §14 is a separate narrative log, with its own three-digit
numbering — it does not index the register, it points to it).

Sites are given by file and by symbol, without line numbers: those drift
with every edit, whereas a `grep` for the symbol stays true.

## Inventory — 16 patterns

| Pattern | Why it exists | Implementation site |
|---|---|---|
| **Decorator** | Adds the cross-cutting concerns — fast path N ≤ 93, observer adaptation, GC control, FFT cache configuration, pool warm-up — without touching the algorithm cores | `FibCalculator` wrapping `CoreCalculator`, `internal/fibonacci/calculator.go` |
| **Strategy** | Lets the multiplication policy be swapped according to the workload or the measurement intent | `Multiplier` / `DoublingStepExecutor` with `AdaptiveStrategy` and `FFTOnlyStrategy`, `internal/fibonacci/strategy.go`; calculators in `fastdoubling.go`, `matrix.go`, `fft_based.go` |
| **Interface Segregation (ISP)** | A consumer that only needs to multiply/square depends on the narrow interface; only the loop frameworks take the wide one | `Multiplier` (narrow: `Multiply`/`Square`) vs `DoublingStepExecutor` (wide: `+ExecuteStep`), `internal/fibonacci/strategy.go` |
| **Factory + Registry** | Centralized registration, lookup and caching of calculators, with lazy creation and double-checking under lock | `DefaultFactory` (`internal/fibonacci/registry.go`), consumed via `orchestration.CalculatorSource` and `app.CalculatorRegistry` — interfaces defined on the consumer side since 2026-09-07 |
| **Observer** | Decouples progress production from its consumers (UI, log) and admits several at once | `progress.ProgressSubject` + `ProgressObserver` implementations, `internal/progress/observer.go`, `observers.go` |
| **Framework (Template Method)** | The framework owns the loop (bit iteration, progress reporting, context checks) and delegates the operation to a pluggable strategy | `DoublingFramework` (`internal/fibonacci/doubling_framework.go`), `MatrixFramework` (`matrix_framework.go`) |
| **Facade** | `app.Application` hides flag parsing, mode dispatch and the error → exit code translation | `internal/app/app.go`, type `Application` and method `Run` |
| **Adapter** | Two uses: a function promoted to `ProgressReporter` without a dedicated type; the `gopsutil` host probes converted into a Bubble Tea message | `orchestration.ProgressReporterFunc` (`internal/orchestration/interfaces.go`) wrapping `cli.DisplayProgress`; `sampleSysStatsCmd` (`internal/tui/commands.go`, folded in from the former `internal/metrics/system`) |
| **Object Pool** | Reduces allocations and GC pressure on hot paths | `sync.Pool` of Fibonacci state (`internal/fibonacci/fastdoubling.go`, `AcquireStateForN`/`ReleaseStateWithResult`) and the per-size-class pools of `internal/bigfft/pool.go`. Capped by `MaxPooledBitLen` = 50,000,000 bits (`internal/fibonacci/common.go`): beyond it, the object is dropped instead of recycled |
| **Arena Allocator** | Pre-sizes a contiguous block for the backing arrays of the state `big.Int`s, which reduces fragmentation and per-buffer GC tracking | `memory.CalculationArena`, `internal/fibonacci/memory/arena.go` (`PreSizeFromArena`) |
| **Bump Allocator** | Batched temporary allocations with O(1) reset for the FFT internals | `bigfft.BumpAllocator`, `internal/bigfft/bump.go` |
| **Cache (LRU) — FFT transforms** | Reuses a forward transform (`PolValues`) from one operation to the next. Bounded in entries **and** in bytes (audit M-08) | `internal/bigfft/fft_cache.go`, type `TransformCache`. ⚠ Consulted only from `Mul`/`MulTo`/`Sqr`/`SqrTo` — **no doubling loop reads it** (`executeDoublingStepFFT` calls `TransformWithBump`); see the `TransformCache` note in the [component-diagram](../component-diagram.md) |
| **Circuit Breaker (lightweight)** | Exits before the OOM rather than during it: the requirement is estimated and compared with the budget before any computation | Estimate `memory.EstimateMemoryUsage` (`internal/fibonacci/memory/budget.go`); the actual exit is `Application.validateMemoryBudget` (`internal/app/calculate.go`), which returns `ExitErrorConfig` |
| **Zero-Copy Result Return** | "Steals" `res.a` from the matrix state instead of copying it | `MatrixFramework.ExecuteMatrixLoop` **only**, `internal/fibonacci/matrix_framework.go`. Deliberately **not** done in `DoublingFramework.ExecuteDoublingLoop` (P1-04): its state aliases the arena, so the success path deep-copies via `ReleaseStateWithResult` |
| **Generics with Pointer Constraints** | A single task execution for multiplications and squarings, without duplication | `executeTasks[T any, PT interface{*T; task}]`, `internal/fibonacci/common.go` |
| **GC Controller** | Turns the GC off during large computations (N ≥ 1M in `auto` mode) and restores it afterwards; `debug.SetMemoryLimit` serves as an OOM safety net | `memory.GCController`, `internal/fibonacci/memory/gc_control.go` (`WithGC`, `Begin`, `End`) |

## Engineering mechanisms — 5

Not design patterns in the catalog sense, but recurring choices a reader
meets throughout the code.

| Mechanism | Detail | Site |
|---|---|---|
| **Run-time configurable threshold heuristics** | Adaptive estimate derived from the core count, the architecture and the SIMD class | `internal/config/thresholds.go`, `hardware.go` |
| **Progress aggregation through a buffered channel** | Buffer = `numCalcs × ProgressBufferMultiplier`, with `ProgressBufferMultiplier = 5` | `internal/orchestration/orchestrator.go` |
| **Lock-free observer snapshot** | `Freeze()` copies the observer slice once and returns a `ProgressCallback` closure: no more lock acquisition in the hot loop | `ProgressSubject.Freeze`, `internal/progress/observer.go` |
| **Semaphore-based concurrency limiting** | **Two** distinct semaphores, sized differently: the one for Fibonacci tasks is `runtime.GOMAXPROCS(0)`, the one for FFT recursion is `runtime.NumCPU()` | `getTaskSemaphore` (`internal/fibonacci/common.go`), `getSemaphore` (`internal/bigfft/fft_recursion.go`) |
| **Functional options** | Construction of the `Application` through options, including the injection of a calculator factory for tests | `AppOption`, `WithFactory`, `internal/app/app.go` |

## Cross-cutting contracts

- **Concurrency**: goroutines always have a bounded lifecycle; error
  propagation goes through `errgroup`, or through the allocation-free `parallel3Result` struct on
  the fastdoubling hot path (`internal/fibonacci/common.go`).
- **Resource ownership**: every state acquisition is paired with a release
  in the same scope (`AcquireStateForN`/`ReleaseStateWithResult`, or the
  per-calculator variants `acquireStateForN`/`releaseStateWithResult`). Since commit `fa13bfd`, the
  release sink is either the shared `sync.Pool` or a GC-immune cache slot
  owned by the calculator (`FastDoublingCalculator.cachedState`, bounded by
  `maxCachedArenaWords`) that keeps the state from one call to the next. A bump arena must
  call `Reset` before reuse. When a cached or pooled state owns an arena,
  every `*big.Int` slot must be detached (`s.FK = new(big.Int)` and so on) before
  the state reaches either sink — otherwise the arena would alias data that the
  next occupant overwrites (`clearStateAliases`, called unconditionally by
  `finalizeStateReleaseTo` in `internal/fibonacci/fastdoubling.go`).

## Maintenance notes

- This file documents the *current* code; it is not a catalog of intentions. An
  added pattern is declared here, with its site.
- A pattern removed from the code is removed from here. There is no second table to
  keep in sync: [`ARCH.md` §5](../../ARCH.md#5-design-patterns) cites only the five
  patterns its own narration depends on, and points to this file for the rest.
- For relations between packages, see [`dependency-graph.md`](../dependency-graph.md);
  for interfaces, [`interface-hierarchy.md`](./interface-hierarchy.md).

---
[← Back to the architecture hub](../README.md) · Narrative legend: [`ARCH.md` §5](../../ARCH.md#5-design-patterns)
