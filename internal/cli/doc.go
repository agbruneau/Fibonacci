// Package cli provides the user-facing command-line presentation layer
// for fibcalc. It is invoked by internal/app once flag parsing has
// produced an internal/config.AppConfig and a calculator selection.
//
// Responsibilities:
//
//   - Format calculation results (full value, last digits, summary)
//     via Presenter implementations of orchestration.ResultPresenter.
//   - Render in-flight progress (spinner, percent, ETA, throughput)
//     for the non-TUI mode. The TUI mode lives in internal/tui and
//     bypasses these helpers.
//   - Emit completion scripts for bash/zsh/fish/powershell — see the
//     internal/cli/completion subpackage for the registry-driven
//     generators and their hardened escape contracts.
//
// Dependencies (downward):
//
//	internal/cli → internal/orchestration, internal/format, internal/config,
//	               internal/progress, internal/errors
//
// This package MUST NOT import internal/fibonacci directly ; calculator
// types arrive through the orchestration.Calculator alias. The
// arch_test.go architecture gate enforces this invariant on the tui
// sibling ; the cli production code follows the same rule by
// convention.
package cli
