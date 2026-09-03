package progress

import (
	"testing"
	"time"
)

// TestChannelObserverTerminalUpdateIsNotDropped pins audit L-03: intermediate
// progress may be dropped when the channel is full, because another update
// follows. The terminal 1.0 has no successor, and consumers read it as "the
// run completed" — cli.DisplayProgress prints "ETA: N/A (interrupted)" when
// the last average it saw is below 1.0, so losing it reported a successful
// calculation as an interrupted one.
func TestChannelObserverTerminalUpdateIsNotDropped(t *testing.T) {
	t.Parallel()

	// Capacity 1, pre-filled: any non-blocking send would be dropped.
	ch := make(chan ProgressUpdate, 1)
	ch <- ProgressUpdate{CalculatorIndex: 0, Value: 0.1}

	obs := NewChannelObserver(ch)

	done := make(chan struct{})
	go func() {
		obs.Update(0, 1.0)
		close(done)
	}()

	// The observer must still be blocked: nothing has drained the channel.
	select {
	case <-done:
		t.Fatal("the terminal update was dropped instead of waiting for room")
	case <-time.After(20 * time.Millisecond):
	}

	// Drain, which frees the slot the blocked send needs.
	if got := <-ch; got.Value != 0.1 {
		t.Fatalf("first queued update = %v, want 0.1", got.Value)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the terminal update never went through after the channel drained")
	}

	got := <-ch
	if got.Value != 1.0 {
		t.Errorf("terminal update = %v, want 1.0", got.Value)
	}
}

// TestChannelObserverIntermediateUpdateIsDroppable is the companion: a slow
// consumer must not be able to throttle the calculation.
func TestChannelObserverIntermediateUpdateIsDroppable(t *testing.T) {
	t.Parallel()

	ch := make(chan ProgressUpdate, 1)
	ch <- ProgressUpdate{CalculatorIndex: 0, Value: 0.1}

	obs := NewChannelObserver(ch)

	done := make(chan struct{})
	go func() {
		obs.Update(0, 0.5) // full channel, no successor guarantee needed
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("an intermediate update must be dropped, not block the caller")
	}
}
