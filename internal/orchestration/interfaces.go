package orchestration

import (
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/progress"
)

// Re-exported aliases of fibonacci package types. UI/presentation packages
// (e.g. internal/tui) should consume these aliases via the orchestration
// import rather than reaching into internal/fibonacci directly — this keeps
// the dependency arrow ui → orchestration → fibonacci. That hierarchy used to
// be asserted in CLAUDE.md, which was removed from the repo on 2026-07-31
// (ADR-0009 status note); what enforces it now is TestArchitectureLayering in
// internal/arch_test.go, not a document.
type (
	// Calculator aliases fibonacci.Calculator for downstream packages.
	Calculator = fibonacci.Calculator

	// Options aliases fibonacci.Options for downstream packages.
	Options = fibonacci.Options
)

// ExecutionConfig groups the parameters required to execute Fibonacci calculations.
type ExecutionConfig struct {
	// Calculators is a slice of calculators to execute.
	Calculators []fibonacci.Calculator
	// N is the Fibonacci index to compute.
	N uint64
	// Opts contains calculation options (thresholds, etc.).
	Opts fibonacci.Options
	// ProgressReporter is used for displaying updates.
	ProgressReporter ProgressReporter
	// Out is the io.Writer for displaying progress updates.
	Out io.Writer
}

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
// It serves as the shared domain type between orchestration and presentation layers.
type CalculationResult struct {
	// Name is the identifier of the algorithm used (e.g., "Fast Doubling").
	Name string
	// Result is the computed Fibonacci number. It is nil if an error occurred.
	Result *big.Int
	// Duration is the time taken to complete the calculation.
	Duration time.Duration
	// Err contains any error that occurred during the calculation.
	Err error
}

// PresentationOptions configures how results are presented to the user.
type PresentationOptions struct {
	N         uint64
	Verbose   bool
	Details   bool
	ShowValue bool
}

// ProgressReporter defines the interface for displaying calculation progress.
// This interface decouples the orchestration layer from the presentation layer,
// following Clean Architecture principles where business logic should not
// depend on UI concerns.
//
// Implementations handle the visual representation of progress (spinners,
// progress bars, etc.) while the orchestration layer focuses on coordinating
// the calculations.
type ProgressReporter interface {
	// DisplayProgress starts displaying progress updates from the channel.
	// It should be called in a separate goroutine and will run until the
	// progressChan is closed.
	//
	// Parameters:
	//   - wg: A WaitGroup to signal when display is complete.
	//   - progressChan: Channel receiving progress updates from calculators.
	//   - numCalculators: The number of concurrent calculators being tracked.
	//   - out: The writer for progress output.
	DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer)
}

// ProgressReporterFunc is a function adapter that implements ProgressReporter.
// This allows passing a function directly where a ProgressReporter is expected.
type ProgressReporterFunc func(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer)

// DisplayProgress calls the underlying function.
func (f ProgressReporterFunc) DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, numCalculators int, out io.Writer) {
	f(wg, progressChan, numCalculators, out)
}

// NullProgressReporter is a no-op implementation of ProgressReporter.
// It drains the progress channel without displaying anything.
// Useful for quiet mode or testing.
type NullProgressReporter struct{}

// DisplayProgress drains the channel without output.
func (NullProgressReporter) DisplayProgress(wg *sync.WaitGroup, progressChan <-chan progress.ProgressUpdate, _ int, _ io.Writer) {
	defer wg.Done()
	for range progressChan {
		// Drain channel silently
	}
}

// ResultPresenter defines the interface for presenting calculation results.
// This interface decouples the orchestration layer from presentation concerns,
// allowing different output formats (CLI, JSON, etc.) without modifying
// the orchestration logic.
type ResultPresenter interface {
	// PresentComparisonTable displays the comparison summary table.
	PresentComparisonTable(results []CalculationResult, out io.Writer)

	// PresentResult displays the final calculation result.
	PresentResult(result CalculationResult, n uint64, verbose, details, showValue bool, out io.Writer)
}

// ErrorHandler handles calculation errors and returns exit codes.
type ErrorHandler interface {
	HandleError(err error, duration time.Duration, out io.Writer) int
}
