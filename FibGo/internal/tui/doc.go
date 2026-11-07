// Package tui provides an interactive terminal user interface (TUI) dashboard
// for monitoring Fibonacci calculations in real-time.
//
// The TUI is inspired by btop and uses the Elm architecture via bubbletea.
// It displays calculation progress, runtime metrics, and results in a
// multi-panel layout with keyboard navigation.
//
// The TUI integrates with the existing orchestration layer through
// implementations of [orchestration.ProgressReporter] and
// [orchestration.ResultPresenter], allowing it to be used as a drop-in
// replacement for the CLI presentation layer.
//
// Activation: pass --tui on the command line or set FIBCALC_TUI=true.
package tui
