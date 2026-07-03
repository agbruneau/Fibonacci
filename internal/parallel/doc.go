// Package parallel provides small, focused primitives for coordinating
// short-lived goroutines inside the Fibonacci calculators.
//
// # Role
//
// The package deliberately contains no goroutine pools or schedulers — that
// responsibility lives in the calculator packages (fastdoubling, matrix,
// fft). What belongs here are the reusable, dependency-free building
// blocks used by many call sites:
//
//   - ErrorCollector: records the first non-nil error observed across N
//     goroutines (sync.Once semantics), so that wg.Wait followed by
//     ec.Err() yields deterministic error propagation.
//
// # Invariants
//
//   - Concurrent writers are safe (e.g. ErrorCollector.SetError); results
//     must be read only after those writers finish (see ErrorCollector.Err).
//   - No exported function spawns goroutines; callers own goroutine
//     lifecycle and are responsible for waiting on completion before
//     reading results.
//
// # Example
//
//	var ec parallel.ErrorCollector
//	var wg sync.WaitGroup
//	for _, task := range tasks {
//	    wg.Add(1)
//	    go func(t Task) {
//	        defer wg.Done()
//	        ec.SetError(t.Run(ctx))
//	    }(task)
//	}
//	wg.Wait()
//	if err := ec.Err(); err != nil {
//	    return err
//	}
package parallel
