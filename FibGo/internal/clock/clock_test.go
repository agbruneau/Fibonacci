package clock

import (
	"testing"
	"time"
)

// TestResolution is the reason the package exists: on Windows the runtime
// clock steps by ~0.5 ms, so its smallest observable step fails this bound.
func TestResolution(t *testing.T) {
	t.Parallel()
	smallest := time.Duration(1<<63 - 1)
	for range 5 {
		s := Now()
		var d time.Duration
		for d == 0 {
			d = Since(s)
		}
		smallest = min(smallest, d)
	}
	if smallest > 50*time.Microsecond {
		t.Fatalf("smallest observable step = %v, want <= 50µs", smallest)
	}
}

func TestSinceTracksSleep(t *testing.T) {
	t.Parallel()
	s := Now()
	time.Sleep(20 * time.Millisecond)
	d := Since(s)
	// Sleep is timed on the runtime clock, which may round down by one step.
	if d < 19*time.Millisecond || d > 2*time.Second {
		t.Fatalf("Since after a 20ms sleep = %v, want within [19ms, 2s]", d)
	}
}

func TestSinceFutureStampIsZero(t *testing.T) {
	t.Parallel()
	if d := Since(Stamp{ticks: ticks() + int64(time.Hour)}); d != 0 {
		t.Fatalf("Since(future) = %v, want 0", d)
	}
}
