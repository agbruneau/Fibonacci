package progress_test

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/agbruneau/FibGo/internal/progress"
)

// ─────────────────────────────────────────────────────────────────────────────
// ProgressSubject Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestNewProgressSubject verifies subject construction.
func TestNewProgressSubject(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	if subject == nil {
		t.Fatal("NewProgressSubject returned nil")
	}
	if subject.ObserverCount() != 0 {
		t.Errorf("new subject should have 0 observers, got %d", subject.ObserverCount())
	}
}

// TestProgressSubject_Register verifies observer registration.
func TestProgressSubject_Register(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()

	// Register nil should be no-op
	subject.Register(nil)
	if subject.ObserverCount() != 0 {
		t.Errorf("registering nil should not add observer, got %d", subject.ObserverCount())
	}

	// Register valid observer
	observer := progress.NewNoOpObserver()
	subject.Register(observer)
	if subject.ObserverCount() != 1 {
		t.Errorf("expected 1 observer, got %d", subject.ObserverCount())
	}

	// Register second observer
	subject.Register(progress.NewNoOpObserver())
	if subject.ObserverCount() != 2 {
		t.Errorf("expected 2 observers, got %d", subject.ObserverCount())
	}
}

// TestProgressSubject_Unregister verifies observer removal.
func TestProgressSubject_Unregister(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	// Use mockObserver instead of NoOpObserver because empty structs
	// may share the same address, breaking pointer comparison
	observer1 := newMockObserver()
	observer2 := newMockObserver()

	subject.Register(observer1)
	subject.Register(observer2)

	if subject.ObserverCount() != 2 {
		t.Fatalf("expected 2 observers, got %d", subject.ObserverCount())
	}

	// Unregister nil should be no-op
	subject.Unregister(nil)
	if subject.ObserverCount() != 2 {
		t.Errorf("unregistering nil should not remove observer, got %d", subject.ObserverCount())
	}

	// Unregister first observer
	subject.Unregister(observer1)
	if subject.ObserverCount() != 1 {
		t.Errorf("expected 1 observer after unregister, got %d", subject.ObserverCount())
	}

	// Unregister non-existent observer should be no-op
	subject.Unregister(observer1)
	if subject.ObserverCount() != 1 {
		t.Errorf("unregistering non-existent should not change count, got %d", subject.ObserverCount())
	}

	// Unregister remaining observer
	subject.Unregister(observer2)
	if subject.ObserverCount() != 0 {
		t.Errorf("expected 0 observers after unregister, got %d", subject.ObserverCount())
	}
}

// mockObserver tracks updates for testing.
type mockObserver struct {
	updates []struct {
		calcIndex int
		progress  float64
	}
	mu sync.Mutex
}

func newMockObserver() *mockObserver {
	return &mockObserver{
		updates: make([]struct {
			calcIndex int
			progress  float64
		}, 0),
	}
}

func (m *mockObserver) Update(calcIndex int, progress float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, struct {
		calcIndex int
		progress  float64
	}{calcIndex, progress})
}

func (m *mockObserver) updateCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.updates)
}

// TestProgressSubject_Notify verifies notification delivery.
func TestProgressSubject_Notify(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	mock1 := newMockObserver()
	mock2 := newMockObserver()

	subject.Register(mock1)
	subject.Register(mock2)

	// Notify with test values
	subject.Notify(1, 0.5)
	subject.Notify(2, 1.0)

	// Verify both observers received updates
	if mock1.updateCount() != 2 {
		t.Errorf("mock1 expected 2 updates, got %d", mock1.updateCount())
	}
	if mock2.updateCount() != 2 {
		t.Errorf("mock2 expected 2 updates, got %d", mock2.updateCount())
	}

	// Verify update values
	if mock1.updates[0].calcIndex != 1 || mock1.updates[0].progress != 0.5 {
		t.Errorf("unexpected first update: %+v", mock1.updates[0])
	}
	if mock1.updates[1].calcIndex != 2 || mock1.updates[1].progress != 1.0 {
		t.Errorf("unexpected second update: %+v", mock1.updates[1])
	}
}

// TestProgressSubject_ConcurrentAccess verifies thread safety.
func TestProgressSubject_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	var wg sync.WaitGroup

	// Concurrent registration
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			subject.Register(progress.NewNoOpObserver())
		}()
	}

	// Concurrent notification
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			subject.Notify(idx, float64(idx)/10.0)
		}(i)
	}

	wg.Wait()

	// Should have 10 observers registered
	if subject.ObserverCount() != 10 {
		t.Errorf("expected 10 observers, got %d", subject.ObserverCount())
	}
}

// TestProgressSubject_AsProgressCallback verifies adapter function.
func TestProgressSubject_AsProgressCallback(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	mock := newMockObserver()
	subject.Register(mock)

	// Get reporter for calcIndex 5
	reporter := subject.AsProgressCallback(5)

	// Call reporter
	reporter(0.25)
	reporter(0.75)

	// Verify updates received
	if mock.updateCount() != 2 {
		t.Errorf("expected 2 updates, got %d", mock.updateCount())
	}
	if mock.updates[0].calcIndex != 5 || mock.updates[0].progress != 0.25 {
		t.Errorf("unexpected update: %+v", mock.updates[0])
	}
}

// TestProgressSubject_FreezeSnapshot verifies Freeze creates an isolated snapshot.
func TestProgressSubject_FreezeSnapshot(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	mock1 := newMockObserver()
	mock2 := newMockObserver()

	subject.Register(mock1)

	// Freeze with only mock1 registered
	callback := subject.Freeze(0)

	// Register mock2 after freeze — should NOT receive updates via callback
	subject.Register(mock2)

	callback(0.5)

	if mock1.updateCount() != 1 {
		t.Errorf("mock1 expected 1 update from frozen callback, got %d", mock1.updateCount())
	}
	if mock2.updateCount() != 0 {
		t.Errorf("mock2 should not receive updates from frozen callback, got %d", mock2.updateCount())
	}
}

// TestProgressSubject_FreezeRaceConditions verifies Freeze is safe during concurrent register/unregister.
func TestProgressSubject_FreezeRaceConditions(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()
	mock := newMockObserver()
	subject.Register(mock)

	var wg sync.WaitGroup

	// Concurrent freeze calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb := subject.Freeze(0)
			cb(0.5)
		}()
	}

	// Concurrent register/unregister during freeze
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			obs := newMockObserver()
			subject.Register(obs)
			subject.Unregister(obs)
		}()
	}

	wg.Wait()
}

// TestProgressSubject_FreezePanicRecovery verifies Freeze recovers from panicking observers.
func TestProgressSubject_FreezePanicRecovery(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()

	// Register a panicking observer
	panicker := &panicObserver{}
	subject.Register(panicker)

	// Register a normal observer after the panicker
	mock := newMockObserver()
	subject.Register(mock)

	callback := subject.Freeze(0)

	before := progress.RecoveredObserverCount()

	// Should not panic — the panicking observer is recovered
	callback(0.5)

	// The mock observer (registered after panicker) should still receive the update
	if mock.updateCount() != 1 {
		t.Errorf("mock expected 1 update despite panicking observer, got %d", mock.updateCount())
	}

	// The recovered-panic counter is process-wide and monotonic, so assert
	// the delta rather than an absolute value.
	if got := progress.RecoveredObserverCount() - before; got != 1 {
		t.Errorf("RecoveredObserverCount() delta = %d, want 1", got)
	}
}

// panicObserver always panics on Update.
type panicObserver struct{}

func (p *panicObserver) Update(int, float64) {
	panic("test panic")
}

// ─────────────────────────────────────────────────────────────────────────────
// ChannelObserver Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestChannelObserver_Update verifies channel updates.
func TestChannelObserver_Update(t *testing.T) {
	t.Parallel()

	ch := make(chan progress.ProgressUpdate, 10)
	observer := progress.NewChannelObserver(ch)

	observer.Update(1, 0.5)
	observer.Update(2, 1.0)

	// Verify updates sent
	update1 := <-ch
	if update1.CalculatorIndex != 1 || update1.Value != 0.5 {
		t.Errorf("unexpected update1: %+v", update1)
	}

	update2 := <-ch
	if update2.CalculatorIndex != 2 || update2.Value != 1.0 {
		t.Errorf("unexpected update2: %+v", update2)
	}
}

// TestChannelObserver_NilChannel verifies nil handling.
func TestChannelObserver_NilChannel(t *testing.T) {
	t.Parallel()

	observer := progress.NewChannelObserver(nil)
	// Should not panic
	observer.Update(1, 0.5)
}

// TestChannelObserver_FullChannel verifies non-blocking behavior.
func TestChannelObserver_FullChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan progress.ProgressUpdate) // Unbuffered = full
	observer := progress.NewChannelObserver(ch)

	// Should not block - use timeout to verify
	done := make(chan bool, 1)
	go func() {
		observer.Update(1, 0.5)
		done <- true
	}()

	// Wait a bit for the goroutine to complete
	// If Update blocks, this will timeout
	select {
	case <-done:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("observer.Update should not block on full channel")
	}
}

// TestChannelObserver_ClampsProgress verifies progress clamping.
func TestChannelObserver_ClampsProgress(t *testing.T) {
	t.Parallel()

	ch := make(chan progress.ProgressUpdate, 1)
	observer := progress.NewChannelObserver(ch)

	observer.Update(1, 1.5) // Should be clamped to 1.0

	update := <-ch
	if update.Value != 1.0 {
		t.Errorf("expected progress clamped to 1.0, got %f", update.Value)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// LoggingObserver Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestLoggingObserver_Update verifies logging behavior.
func TestLoggingObserver_Update(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	observer := progress.NewLoggingObserver(logger, 0.1)

	// First update should log (initial progress)
	observer.Update(0, 0.1)
	if buf.Len() == 0 {
		t.Error("expected initial progress to be logged")
	}

	// Small increase should not log (below threshold)
	buf.Reset()
	observer.Update(0, 0.15)
	if buf.Len() > 0 {
		t.Error("expected small progress change to not be logged")
	}

	// Large increase should log
	buf.Reset()
	observer.Update(0, 0.5)
	if buf.Len() == 0 {
		t.Error("expected significant progress change to be logged")
	}

	// Completion should log
	buf.Reset()
	observer.Update(0, 1.0)
	if buf.Len() == 0 {
		t.Error("expected completion to be logged")
	}
}

// TestLoggingObserver_DefaultThreshold verifies default threshold handling.
func TestLoggingObserver_DefaultThreshold(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)

	// Zero threshold should use default
	observer := progress.NewLoggingObserver(logger, 0)

	observer.Update(0, 0.0)
	observer.Update(0, 0.05) // Below default 0.1 threshold
	if buf.Len() == 0 {
		t.Error("expected first update to be logged")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NoOpObserver Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestNoOpObserver_Update verifies no-op behavior.
func TestNoOpObserver_Update(t *testing.T) {
	t.Parallel()

	observer := progress.NewNoOpObserver()
	// Should not panic
	observer.Update(0, 0.0)
	observer.Update(1, 1.0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Integration Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestMultipleObserversIntegration verifies multiple observers work together.
func TestMultipleObserversIntegration(t *testing.T) {
	t.Parallel()

	subject := progress.NewProgressSubject()

	// Set up channel observer
	ch := make(chan progress.ProgressUpdate, 10)
	channelObs := progress.NewChannelObserver(ch)

	// Set up logging observer
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).Level(zerolog.DebugLevel)
	loggingObs := progress.NewLoggingObserver(logger, 0.1)

	// Set up mock observer to count
	var updateCount int64
	countingObs := &struct{ progress.ProgressObserver }{} // Anonymous observer
	countingObsImpl := newMockObserver()

	subject.Register(channelObs)
	subject.Register(loggingObs)
	subject.Register(countingObsImpl)

	// Notify progress
	subject.Notify(0, 0.5)
	subject.Notify(0, 1.0)

	// Verify channel received updates
	if len(ch) != 2 {
		t.Errorf("expected 2 channel updates, got %d", len(ch))
	}

	// Verify mock received updates
	if countingObsImpl.updateCount() != 2 {
		t.Errorf("expected 2 mock updates, got %d", atomic.LoadInt64(&updateCount))
	}

	// Verify no panics occurred (implicit)
	_ = countingObs
}
