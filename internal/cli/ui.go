package cli

import (
	"time"

	"github.com/briandowns/spinner"
)

const (
	// TruncationLimit is the digit threshold from which a result is truncated
	// in standard output to avoid cluttering the terminal.
	TruncationLimit = 100
	// DisplayEdges specifies the number of digits to display at the beginning
	// and end of a truncated number.
	DisplayEdges = 25
	// HexDisplayEdges specifies the number of hex characters to display at the
	// beginning and end of a truncated hexadecimal number.
	HexDisplayEdges = 40
	// ProgressRefreshRate defines the refresh frequency of the progress bar.
	// Optimized to 200ms to reduce updates and improve performance.
	ProgressRefreshRate = 200 * time.Millisecond
	// ProgressBarWidth defines the width in characters of the progress bar.
	ProgressBarWidth = 40
)

// Spinner is an interface that abstracts the behavior of a terminal spinner.
// This allows for the decoupling of the `DisplayProgress` function from a
// specific spinner implementation, facilitating easier testing and maintenance.
// It defines the essential controls for a spinner: starting, stopping, and
// updating its status message.
type Spinner interface {
	// Start begins the spinner animation.
	Start()
	// Stop halts the spinner animation.
	Stop()
	// UpdateSuffix sets the text that is displayed after the spinner.
	//
	// Parameters:
	//   - suffix: The text string to display.
	UpdateSuffix(suffix string)
}

// realSpinner is a wrapper for the `spinner.Spinner` that implements the
// `Spinner` interface. This adapter allows the `spinner` library to be used
// within the application's CLI framework.
//
// stop/start/setSuffix default to the wrapped spinner's own methods/field
// and only exist as indirection so tests can assert the ordering below
// without a real terminal (see ui_suffix_race_test.go).
type realSpinner struct {
	s *spinner.Spinner

	stop      func()
	start     func()
	setSuffix func(string)
}

// Start begins the spinner animation.
func (rs *realSpinner) Start() {
	rs.start()
}

// Stop halts the spinner animation.
func (rs *realSpinner) Stop() {
	rs.stop()
}

// UpdateSuffix sets the text that is displayed after the spinner.
//
// The spinner library's render goroutine reads Suffix under its internal
// mutex while running; writing rs.s.Suffix directly from here would race
// with it in a real terminal (CONC-01). Stopping the spinner first blocks
// until that goroutine has exited (Stop() happens-before the write), and
// starting it again afterwards spawns a fresh goroutine that only reads
// Suffix after the write (write happens-before Start()) -- no concurrent
// access is possible.
//
// Parameters:
//   - suffix: The string to display.
func (rs *realSpinner) UpdateSuffix(suffix string) {
	rs.stop()
	rs.setSuffix(suffix)
	rs.start()
}

var newSpinner = func(options ...spinner.Option) Spinner {
	// Using the same interval as ProgressRefreshRate to synchronize
	s := spinner.New(spinner.CharSets[11], ProgressRefreshRate, options...)
	rs := &realSpinner{s: s}
	rs.stop = s.Stop
	rs.start = s.Start
	rs.setSuffix = func(suffix string) { s.Suffix = suffix }
	return rs
}
