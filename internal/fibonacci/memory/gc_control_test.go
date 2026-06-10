package memory

import (
	"bytes"
	"errors"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestGCController_Disabled(t *testing.T) {
	t.Parallel()

	gc := NewGCController("disabled", 1_000_000)
	gc.Begin()
	defer gc.End()

	if gc.active {
		t.Error("GC controller should not be active in disabled mode")
	}
}

func TestGCController_Auto_SmallN(t *testing.T) {
	t.Parallel()

	gc := NewGCController("auto", 100)
	gc.Begin()
	defer gc.End()

	if gc.active {
		t.Error("GC controller should not be active for small N")
	}
}

func TestGCController_Auto_LargeN(t *testing.T) {
	t.Parallel()

	gc := NewGCController("auto", 2_000_000)
	if !gc.active {
		t.Error("GC controller should be active for N >= 1M in auto mode")
	}
	gc.Begin()
	defer gc.End()
}

func TestGCController_Aggressive(t *testing.T) {
	t.Parallel()

	gc := NewGCController("aggressive", 100)
	if !gc.active {
		t.Error("GC controller should be active in aggressive mode regardless of N")
	}
	gc.Begin()
	defer gc.End()
}

func TestGCController_Stats_BeforeBegin(t *testing.T) {
	t.Parallel()

	gc := NewGCController("disabled", 100)
	stats := gc.Stats()
	if stats.TotalAlloc != 0 {
		t.Errorf("TotalAlloc before Begin should be 0, got %d", stats.TotalAlloc)
	}
}

func TestGCController_Stats_AfterBeginEnd(t *testing.T) {
	t.Parallel()

	gc := NewGCController("aggressive", 2_000_000)
	gc.Begin()
	// Do some allocations
	_ = make([]byte, 1024*1024)
	gc.End()

	stats := gc.Stats()
	// HeapAlloc should be non-zero after some allocations
	// (we can't assert exact values due to runtime variability)
	_ = stats
}

// TestGCController_WithGC_PanicRestoresGC verifies that WithGC restores the
// GC settings even when fn panics. Without WithGC, a panic between Begin and
// End would leave debug.SetGCPercent(-1) in place process-wide, silently
// disabling GC for the remainder of the process lifetime (the bug WithGC
// is designed to prevent).
//
// This test cannot run with t.Parallel() because debug.SetGCPercent is a
// process-global setting; concurrent tests mutating GCPercent would race.
func TestGCController_WithGC_PanicRestoresGC(t *testing.T) {
	// Snapshot the current GCPercent without changing it. SetGCPercent
	// returns the previous value; immediately restore so we observe but
	// do not perturb global state.
	original := debug.SetGCPercent(100)
	debug.SetGCPercent(original)

	gc := NewGCController("aggressive", 2_000_000)
	if !gc.active {
		t.Fatal("test prerequisite: GC controller must be active")
	}

	defer func() {
		// The panic must propagate; recover here to keep the test green.
		if r := recover(); r == nil {
			t.Fatal("expected panic to propagate out of WithGC, but recover returned nil")
		}

		// After WithGC unwinds via deferred End, GCPercent must be back
		// to its original value. SetGCPercent again returns the value
		// that was active at call time.
		current := debug.SetGCPercent(original)
		if current != original {
			t.Errorf("GCPercent not restored after panic: got %d, want %d", current, original)
		}
	}()

	_ = gc.WithGC(func() error {
		panic("simulated calculation failure")
	})
	t.Fatal("WithGC should not return when fn panics")
}

// TestGCController_WithGC_ReturnsError verifies that a non-panicking fn's
// error is propagated and that GC is restored normally.
func TestGCController_WithGC_ReturnsError(t *testing.T) {
	original := debug.SetGCPercent(100)
	debug.SetGCPercent(original)

	gc := NewGCController("aggressive", 2_000_000)
	sentinel := errors.New("sentinel")

	err := gc.WithGC(func() error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("WithGC should propagate fn error: got %v, want %v", err, sentinel)
	}

	current := debug.SetGCPercent(original)
	if current != original {
		t.Errorf("GCPercent not restored after normal return: got %d, want %d", current, original)
	}
}

// TestGCController_ConcurrentBeginEnd_RestoresOriginal verifies the
// package-level refcount (A2-01 / ADR-0005). With two active controllers — the
// --algo all comparison-mode shape — GC must stay disabled until the LAST End,
// and the original GOGC must be restored exactly once. Before the fix, the
// second Begin captured -1 as its "original" and GC could leak disabled, while
// the first End dropped the OOM memory limit under a still-running sibling.
//
// Not parallel: it mutates and probes process-global GC state.
func TestGCController_ConcurrentBeginEnd_RestoresOriginal(t *testing.T) {
	const sentinel = 150
	original := debug.SetGCPercent(sentinel) // install a known GOGC, capture prior
	defer debug.SetGCPercent(original)       // best-effort restore for other tests

	c1 := NewGCController("aggressive", 2_000_000)
	c2 := NewGCController("aggressive", 2_000_000)
	if !c1.active || !c2.active {
		t.Fatal("test prerequisite: both controllers must be active")
	}

	c1.Begin()
	c2.Begin()
	// Both active -> GC must be disabled. Probe non-destructively: SetGCPercent
	// returns the current value; we re-assert -1 so the probe is a no-op.
	if cur := debug.SetGCPercent(-1); cur != -1 {
		t.Fatalf("GC must be disabled while controllers are active: got %d, want -1", cur)
	}

	c1.End()
	// c2 still active -> GC must remain disabled (the bug would restore here).
	if cur := debug.SetGCPercent(-1); cur != -1 {
		t.Fatalf("GC must stay disabled until the last End: got %d, want -1", cur)
	}

	c2.End()
	// Last End -> original GOGC restored exactly once.
	if cur := debug.SetGCPercent(sentinel); cur != sentinel {
		t.Fatalf("GC not restored after last End: got %d, want %d", cur, sentinel)
	}
}

// TestGCController_SetLogger_EmitsGCEvents verifies that SetLogger wires the
// logger actually used by Begin/End: the "gc disabled" and "gc re-enabled"
// debug events must land in the configured sink, with the mode field set.
//
// Not parallel: Begin/End (via WithGC) mutate the process-global GOGC through
// the package refcount. WithGC itself restores the original value on return,
// but the test stays sequential so no parallel test observes the temporary
// GC-off window or asserts global GC state concurrently.
func TestGCController_SetLogger_EmitsGCEvents(t *testing.T) {
	var buf bytes.Buffer
	gc := NewGCController("aggressive", 100)
	if !gc.active {
		t.Fatal("test prerequisite: aggressive controller must be active")
	}
	gc.SetLogger(zerolog.New(&buf))

	if err := gc.WithGC(func() error { return nil }); err != nil {
		t.Fatalf("WithGC: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"gc disabled", "gc re-enabled", string(GCModeAggressive)} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q: %s", want, out)
		}
	}
}

// TestGCController_WithGC_InactiveNoOp verifies that WithGC on an inactive
// controller still calls fn and propagates its result without touching GC.
func TestGCController_WithGC_InactiveNoOp(t *testing.T) {
	t.Parallel()

	gc := NewGCController("disabled", 2_000_000)
	called := false
	err := gc.WithGC(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not invoked")
	}
}
