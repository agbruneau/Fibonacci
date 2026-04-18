// Package fibonacci provides implementations for calculating Fibonacci numbers.
// It exposes a `Calculator` interface that abstracts the underlying calculation
// algorithm, allowing different strategies (Fast Doubling, Matrix Exponentiation,
// FFT-based) to be used interchangeably. The package integrates optimizations such
// as memory pooling, parallel processing, and dynamic threshold adjustment.
//
// # Must* convention (panics on error)
//
// Functions prefixed with "Must" (e.g. MustNewCalculator, DefaultFactory.MustGet)
// follow the standard Go convention of panicking on error instead of returning
// an error value. They are intended for two narrow use cases:
//
//  1. Package-level variable initialization where an error is a programming
//     mistake that must surface immediately at process start, not be silently
//     ignored or lazily handled.
//  2. Tests, where panicking is preferable to the noise of error-handling
//     boilerplate and a failure invariably indicates a programmer bug.
//
// Production code paths that can recover from the underlying failure should
// always call the non-Must variant (NewCalculator, Factory.Get) and handle
// the returned error explicitly. See audit P2-13.
package fibonacci
