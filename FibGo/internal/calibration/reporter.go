// This file defines the presentation port of the calibration package.

package calibration

import "time"

// Reporter is how calibration talks to a user. It is defined here, by the
// consumer, and implemented by the adapter that owns the terminal —
// internal/cli (audit ARC-01).
//
// Before this port existed, calibration imported internal/ui and wrote colored
// text directly: 31 ui.Color* calls across calibration.go, io.go and
// strategy_fast.go. That is the application layer doing presentation, which the
// book rules out for the service layer (ch. 14, p. 367: "this layer does not
// talk to HTTP, SQL, Kafka, or file systems directly"). The practical cost was
// visible in this package's own tests, which had to strip ANSI escapes to
// assert on a threshold value.
//
// Four methods, split by what the reader is expected to do about the message,
// not by how it should look. Color, emphasis and layout belong to the adapter.
type Reporter interface {
	// Notice reports normal progress: what was detected, what was loaded, what
	// was decided.
	Notice(format string, args ...any)

	// Warning reports something that went wrong but did not stop the run —
	// a profile that could not be saved, a calibration that fell back to
	// defaults.
	Warning(format string, args ...any)

	// Error reports a condition that ends the calibration.
	Error(format string, args ...any)

	// Summary renders the completed sweep: one row per threshold tried, with
	// best marked. Rendering the table is the adapter's job; the ordering and
	// the choice of best are decided here.
	Summary(results []PassResult, best int)
}

// PassResult is the outcome of one calibration pass: the threshold that was
// tried, how long F(CalibrationN) took at it, and the error if it failed.
//
// Exported because Reporter.Summary takes a slice of them across the package
// boundary. Duration is meaningless when Err is non-nil.
type PassResult struct {
	Threshold int
	Duration  time.Duration
	Err       error
}

// NopReporter discards everything. It is the reporter for callers that want the
// calibration result and none of the narration — tests, and any programmatic
// embedding.
type NopReporter struct{}

func (NopReporter) Notice(string, ...any)     {}
func (NopReporter) Warning(string, ...any)    {}
func (NopReporter) Error(string, ...any)      {}
func (NopReporter) Summary([]PassResult, int) {}
