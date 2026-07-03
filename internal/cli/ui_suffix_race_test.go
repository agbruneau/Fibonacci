package cli

import (
	"testing"
)

// TestUpdateSuffix_StopWriteStartOrder documents and asserts the fix for
// CONC-01 (audit.md M4): the spinner library's render goroutine reads
// s.Suffix under its internal mutex while UpdateSuffix used to write
// rs.s.Suffix directly with no synchronization — a real data race in a
// TTY session. `go test -race` never observes it because Start() only
// spawns the render goroutine when isRunningInTerminal(s) is true, which
// is never the case for the non-TTY writers used by tests (spinner v1.23.2,
// spinner.go:316). So this test asserts the contract of the fix instead:
// UpdateSuffix must stop the spinner, mutate the suffix, then restart it
// -- Stop() happens-before the write, and the write happens-before the
// goroutine spawned by the following Start(), so no concurrent access to
// Suffix is possible regardless of TTY state.
func TestUpdateSuffix_StopWriteStartOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	rs := &realSpinner{
		stop:  func() { calls = append(calls, "stop") },
		start: func() { calls = append(calls, "start") },
		setSuffix: func(suffix string) {
			calls = append(calls, "write:"+suffix)
		},
	}

	rs.UpdateSuffix(" 42%")

	want := []string{"stop", "write: 42%", "start"}
	if len(calls) != len(want) {
		t.Fatalf("UpdateSuffix call order = %v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("call[%d] = %q, want %q (full sequence %v)", i, calls[i], want[i], calls)
		}
	}
}
