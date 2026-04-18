// Package main is a standalone generator for the golden test data used by the
// Fibonacci test suite (internal/fibonacci/testdata/fibonacci_golden.json).
//
// # Independent oracle (P2-04)
//
// This command deliberately carries its own iterative math/big Fibonacci
// implementation (`fibBig` in main.go) instead of importing
// internal/fibonacci.calculateSmall. The duplication is intentional and
// must not be removed:
//
//   - The generator is the oracle that produces the ground-truth values the
//     main library is validated against. Reusing the library to validate the
//     library would turn every golden test into a tautology (“F(n) equals
//     itself”) — any regression in calculator.go would silently flow into
//     the regenerated JSON and be blessed as the new truth on the next run.
//   - Keeping two independent, dead-simple iterators guarantees a cross-check
//     path: if calculator.go.calculateSmall ever drifted from cmd/generate-golden.fibBig,
//     the Fibonacci golden tests (internal/fibonacci/fibonacci_golden_test.go)
//     would flag the divergence immediately.
//   - Both implementations are intentionally trivial (≤ 18 LoC, straight
//     iterative `a, b = b, a+b`). Their algorithmic surface is small enough
//     that "drift from the library" is the scenario worth defending against;
//     independent reimplementation of such a small kernel is the cheapest
//     defense available.
//
// If you touch fibBig, do NOT collapse it into calculateSmall; instead update
// main.go's comment on fibBig and re-run `go run ./cmd/generate-golden` so the
// committed golden file reflects the new reference implementation.
package main
