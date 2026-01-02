package app

import (
	"context"
	"testing"
	"time"
)

// TestSetupContext tests the SetupContext function.
func TestSetupContext(t *testing.T) {
	t.Parallel()
	t.Run("Context is canceled after timeout", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		timeout := 50 * time.Millisecond

		ctxWithTimeout, cancel := SetupContext(ctx, timeout)
		defer cancel()

		// Context should not be done immediately
		select {
		case <-ctxWithTimeout.Done():
			t.Error("Context should not be done immediately")
		default:
			// Expected
		}

		// Wait for timeout
		time.Sleep(100 * time.Millisecond)

		// Context should be done after timeout
		select {
		case <-ctxWithTimeout.Done():
			if ctxWithTimeout.Err() != context.DeadlineExceeded {
				t.Errorf("Expected DeadlineExceeded error, got %v", ctxWithTimeout.Err())
			}
		default:
			t.Error("Context should be done after timeout")
		}
	})

	t.Run("Context can be canceled manually", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		timeout := 1 * time.Hour

		ctxWithTimeout, cancel := SetupContext(ctx, timeout)
		cancel()

		select {
		case <-ctxWithTimeout.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be done after cancel")
		}
	})
}

// TestSetupLifecycle tests the SetupLifecycle function.
func TestSetupLifecycle(t *testing.T) {
	t.Parallel()
	t.Run("Returns context and cleanup functions", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		timeout := 1 * time.Hour

		ctxWithLifecycle, cancelFuncs := SetupLifecycle(ctx, timeout)

		// Context should not be nil
		if ctxWithLifecycle == nil {
			t.Error("Context should not be nil")
		}

		// CancelFuncs should not be nil
		if cancelFuncs == nil {
			t.Error("CancelFuncs should not be nil")
		}

		// Cleanup should work without panic
		cancelFuncs.Cleanup()
	})

	t.Run("Cleanup cancels context", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		timeout := 1 * time.Hour

		ctxWithLifecycle, cancelFuncs := SetupLifecycle(ctx, timeout)
		cancelFuncs.Cleanup()

		select {
		case <-ctxWithLifecycle.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be done after Cleanup")
		}
	})
}

// TestCancelFuncsCleanup tests the CancelFuncs.Cleanup method.
func TestCancelFuncsCleanup(t *testing.T) {
	t.Parallel()
	t.Run("Cleanup with nil functions does not panic", func(t *testing.T) {
		t.Parallel()
		cf := &CancelFuncs{}
		// Should not panic
		cf.Cleanup()
	})

	t.Run("Cleanup calls both functions", func(t *testing.T) {
		t.Parallel()
		timeoutCalled := false
		signalsCalled := false

		cf := &CancelFuncs{
			CancelTimeout: func() { timeoutCalled = true },
			StopSignals:   func() { signalsCalled = true },
		}

		cf.Cleanup()

		if !timeoutCalled {
			t.Error("CancelTimeout should have been called")
		}
		if !signalsCalled {
			t.Error("StopSignals should have been called")
		}
	})
}
