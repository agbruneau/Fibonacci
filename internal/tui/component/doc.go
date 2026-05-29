// Package component documents the contract that every TUI sub-component must
// satisfy. The package intentionally contains only an interface declaration:
// the concrete component implementations (header, footer, logs, chart,
// metrics) live in package tui itself, where they are exercised by the
// existing white-box tests in tui/*_test.go.
//
// Rationale (R3.4 partial extraction):
//   - The original tui package tests (tui/header_test.go, tui/chart_test.go,
//     tui/logs_test.go, tui/metrics_test.go, tui/footer_test.go,
//     tui/model_test.go) reach into unexported component fields
//     (m.chart.averageProgress, m.logs.entries, m.metrics.alloc, etc.) to
//     verify behavior. Relocating the component types to a sub-package
//     would require either re-exporting every internal field or rewriting
//     dozens of tests — both forbidden by the surgical-change directive
//     (CLAUDE.md, FibGo §5).
//   - Instead, R3.4 extracts the *router* concern (per-message handlers,
//     command builders, layout math) out of tui/model.go into focused
//     siblings (handlers.go, commands.go, layout.go). This brings model.go
//     down to a thin router (~140 lines) without breaking any test.
//   - This sub-package preserves the documented architectural target so a
//     future tighter refactor (with a coordinated test rewrite) can move
//     the component types here behind the Component interface declared
//     below.
package component

import tea "github.com/charmbracelet/bubbletea"

// Component is the contract every TUI sub-component (header, footer, logs,
// chart, metrics) implements in spirit today and would implement formally if
// relocated here. A Component owns its local state, processes a slice of the
// incoming message stream, and renders a string view.
//
// Note: today's concrete types in package tui satisfy this contract by
// convention (Update is split into typed methods such as AddProgressEntry
// rather than a single tea.Msg dispatcher) — the router in tui.Model
// performs the type assertion that this interface would otherwise hide.
type Component interface {
	// Update processes an incoming message and returns the (possibly mutated)
	// component along with any follow-up command. Implementations should be
	// pure with respect to their receiver: returning a new value rather than
	// mutating in place keeps the bubbletea Elm-style update contract sound.
	Update(msg tea.Msg) (Component, tea.Cmd)

	// View renders the component's current state as a string. The caller
	// (tui.Model) is responsible for composing component views into the full
	// dashboard layout.
	View() string
}
