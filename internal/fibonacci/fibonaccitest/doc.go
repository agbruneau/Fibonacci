// Package fibonaccitest exposes test doubles for the fibonacci package.
// It exists so unit tests in upstream packages (internal/orchestration,
// internal/app, internal/tui) can construct lightweight calculators
// without reaching into unexported fibonacci internals.
//
// Primary type:
//
//   - CoreStub: a minimal fibonacci.CoreCalculator implementation
//     returning a configurable result and error. Wrap it with
//     fibonacci.NewCalculator (or MustNewCalculator in tests) to get a
//     full Calculator that satisfies orchestration.Calculator.
//
// Usage example:
//
//	stub := fibonaccitest.CoreStub{
//	    NameValue:   "stub",
//	    ResultValue: big.NewInt(42),
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
//   - The stub's Calculate method completes synchronously without
//     spawning goroutines ; concurrent test scenarios remain
//     deterministic.
package fibonaccitest
