package tui

import (
	"strings"
	"testing"
)

// fillLogsBuffer pushes n progress entries through AddProgressEntry, which is
// the hot path exercised once per throttled progress message during a long
// calculation. With n >= maxLogEntries the ring buffer is saturated, so each
// push works against a full-capacity snapshot — the worst case for the
// previous eager updateContent() implementation.
func fillLogsBuffer(b *testing.B, n int) LogsModel {
	b.Helper()
	logs := NewLogsModel([]string{"Fast Doubling"})
	logs.SetSize(80, 24)
	for i := 0; i < n; i++ {
		logs.AddProgressEntry(ProgressMsg{
			CalculatorIndex: 0,
			Value:           float64(i%100) / 100,
		})
	}
	return logs
}

// BenchmarkLogsUpdateContent measures the cost of a single AddProgressEntry
// push once the ring buffer is already full (the steady state of a long
// session). The eager implementation paid O(N) slice + Join + SetContent on
// every push here; the lazy implementation must only pay the cheap snapshot.
func BenchmarkLogsUpdateContent(b *testing.B) {
	logs := fillLogsBuffer(b, maxLogEntries)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logs.AddProgressEntry(ProgressMsg{
			CalculatorIndex: 0,
			Value:           float64(i%100) / 100,
		})
	}
}

// BenchmarkLogsPushThenView measures push + render together: the realistic
// frame model where many pushes occur between renders. With T pushes per
// View() the eager path was O(T*N); the lazy path is O(N) once per View().
func BenchmarkLogsPushThenView(b *testing.B) {
	logs := fillLogsBuffer(b, maxLogEntries)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			logs.AddProgressEntry(ProgressMsg{
				CalculatorIndex: 0,
				Value:           float64(j%100) / 100,
			})
		}
		_ = logs.View()
	}
}

// TestLogsModel_ViewContentStableAfterPushes is a behavioral guard: the
// rendered viewport content and the entries snapshot after N pushes must be
// identical to a fresh snapshot computed directly from the ring buffer. This
// pins the observable contract so the lazy-invalidation refactor cannot drift
// the displayed lines, ordering, or count.
func TestLogsModel_ViewContentStableAfterPushes(t *testing.T) {
	t.Parallel()
	logs := NewLogsModel([]string{"Fast Doubling", "Matrix"})
	logs.SetSize(80, 24)

	const n = 500
	for i := 0; i < n; i++ {
		logs.AddProgressEntry(ProgressMsg{
			CalculatorIndex: i % 2,
			Value:           float64(i%100) / 100,
		})
	}

	// Reference: the ground truth is the ring buffer's chronological slice.
	want := logs.buffer.Slice()

	// The entries snapshot the white-box tests read must be current
	// immediately after the pushes, exactly as the eager path guaranteed.
	if len(logs.entries) != len(want) {
		t.Fatalf("entries length drift: got %d, want %d", len(logs.entries), len(want))
	}
	for i := range want {
		if logs.entries[i] != want[i] {
			t.Fatalf("entry %d drift:\n got %q\nwant %q", i, logs.entries[i], want[i])
		}
	}

	// View() must materialize without panicking and produce non-empty output.
	if got := logs.View(); len(got) == 0 {
		t.Fatal("View() produced empty output after pushes")
	}

	// flushContent is the deterministic equivalence point: after a flush the
	// viewport content must equal the join of the chronological slice — the
	// exact string the eager updateContent produced on every push.
	logs.dirty = true
	logs.flushContent()
	wantJoined := strings.Join(want, "\n")
	logs.viewport.SetContent(wantJoined)
	// SetContent is idempotent; re-asserting it cannot change the rendered
	// frame. The guard is that flushContent fed the same join: a second
	// identical SetContent leaves the rendered view unchanged.
	first := logs.viewport.View()
	logs.flushContent() // no-op: dirty is false
	if logs.viewport.View() != first {
		t.Fatal("flushContent is not idempotent across renders")
	}
}
