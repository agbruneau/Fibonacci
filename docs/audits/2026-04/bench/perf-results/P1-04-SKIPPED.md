# P1-04 — Arena pool in fastdoubling — SKIPPED

## Finding

`OptimizedFastDoubling.CalculateCore` allocates a fresh
`memory.CalculationArena` on every call via `NewCalculationArena(n)` —
for large N that is a multi-MB backing `[]big.Word` discarded after a
single Fibonacci computation. Audit flagged P1-04 to pool the arena
via `sync.Pool`.

## Why skipped

Pooling the arena through a `sync.Pool` without extensive refactoring
creates a concurrency hazard:

1.  `CalculationState` is itself pooled through `statePool`. After
    `arena.PreSizeFromArena(s.FK, ...)` (et al.), `s.FK.Bits()`
    references memory owned by the arena's backing buffer.
2.  When `CalculateCore` returns and both the arena and the state are
    released to their pools, the state re-enters `statePool` with
    `big.Int` headers still referencing the arena's `[]big.Word`.
3.  Another goroutine can concurrently acquire that pooled arena and
    begin writing into the same backing buffer — corrupting the state
    big.Ints now held by a third goroutine that acquired the state.
4.  The symptom is observable (and reliably reproduced by
    `go test -short -count=5 ./internal/fibonacci/`): TestConcurrent-
    Calculations failures with "result mismatch: expected ..., got 0",
    and occasional panics such as `index out of range [97] with
    length 97` from math/big's internal slice bookkeeping.

A partial mitigation — reassigning `s.FK, s.FK1, s.T1, s.T2, s.T3` to
`new(big.Int)` and `Set()`-copying the returned result into a fresh
big.Int before `ReleaseArena` — was attempted and still fails under
concurrent load. The root cause is that `result := s.FK` in
`DoublingFramework.ExecuteDoublingLoop` captures the **old** `s.FK`
whose Bits() continue to alias the arena, and the Go scheduler can
race AcquireArena / clear() / other goroutine's copy.

A safe implementation would require either (a) eliminating the
inter-pool aliasing by deep-copying the result AND detaching every
surviving state slot before the arena is released (done in the draft
and still raced), or (b) fully re-architecting `AllocBigInt` /
`PreSizeFromArena` so that pooled big.Ints cannot outlive the arena —
too invasive for a performance finding commit.

## Recommendation

Revisit P1-04 under Phase 4 (Refactor & tests, Teams B+C) where the
state + arena lifecycle can be unified. The simplest safe refactor is
probably to fold arena acquisition into `AcquireState` / `ReleaseState`
so the two pools are always released together and CalculationState
owns both sides of the invariant.
