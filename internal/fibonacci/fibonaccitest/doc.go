// Package fibonaccitest exposes test doubles for the fibonacci package.
// It exists so unit tests in upstream packages (internal/orchestration,
// internal/app, internal/tui) can construct lightweight calculators
// without reaching into unexported fibonacci internals.
//
// Primary type:
//
//   - CoreStub: a minimal fibonacci.CoreCalculator implementation whose
//     name and CalculateCore behavior are configured via the NameVal and
//     CoreFunc fields. Wrap it with fibonacci.NewCalculator (or
//     MustNewCalculator in tests) to get a full Calculator that satisfies
//     orchestration.Calculator.
//
// Usage example:
//
//	stub := fibonaccitest.CoreStub{
//	    NameVal: "stub",
//	    CoreFunc: func(ctx context.Context, _ progress.ProgressCallback, n uint64, _ fibonacci.Options) (*big.Int, error) {
//	        return big.NewInt(42), nil
//	    },
//	}
//	calc := fibonacci.MustNewCalculator(&stub)
//	results := orchestration.ExecuteCalculations(ctx, orchestration.ExecutionConfig{
//	    Calculators: []orchestration.Calculator{calc},
//	    N:           10,
//	})
//
// Test isolation guarantees:
//
//   - CoreStub does not allocate big.Int buffers from any pool, so test
//     fixtures cannot corrupt the production memory invariants documented
//     in internal/fibonacci/fastdoubling.go.
//   - The stub's CalculateCore method completes synchronously without
//     spawning goroutines ; concurrent test scenarios remain
//     deterministic.
package fibonaccitest
