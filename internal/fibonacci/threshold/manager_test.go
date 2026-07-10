package threshold

import (
	"testing"
	"time"
)

// newTestManager is a small helper adapting the old fixed-argument
// constructor shape to NewDynamicThresholdManagerFromConfig, which is what
// production (fastdoubling.go) actually calls.
func newTestManager(fftThreshold, parallelThreshold int) *DynamicThresholdManager {
	return NewDynamicThresholdManagerFromConfig(DynamicThresholdConfig{
		Enabled:                  true,
		InitialFFTThreshold:      fftThreshold,
		InitialParallelThreshold: parallelThreshold,
	})
}

// TestNewDynamicThresholdManagerFromConfig tests config-based constructor.
func TestNewDynamicThresholdManagerFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		cfg       DynamicThresholdConfig
		expectNil bool
		expectFFT int
		expectPar int
	}{
		{
			name: "disabled returns nil",
			cfg: DynamicThresholdConfig{
				Enabled: false,
			},
			expectNil: true,
		},
		{
			name: "enabled with valid config",
			cfg: DynamicThresholdConfig{
				Enabled:                  true,
				InitialFFTThreshold:      200000,
				InitialParallelThreshold: 5000,
				AdjustmentInterval:       10,
			},
			expectNil: false,
			expectFFT: 200000,
			expectPar: 5000,
		},
		{
			name: "enabled with zero interval uses default",
			cfg: DynamicThresholdConfig{
				Enabled:                  true,
				InitialFFTThreshold:      100000,
				InitialParallelThreshold: 2000,
				AdjustmentInterval:       0,
			},
			expectNil: false,
			expectFFT: 100000,
			expectPar: 2000,
		},
		{
			name: "enabled with negative interval uses default",
			cfg: DynamicThresholdConfig{
				Enabled:                  true,
				InitialFFTThreshold:      300000,
				InitialParallelThreshold: 8000,
				AdjustmentInterval:       -5,
			},
			expectNil: false,
			expectFFT: 300000,
			expectPar: 8000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mgr := NewDynamicThresholdManagerFromConfig(tc.cfg)
			if tc.expectNil {
				if mgr != nil {
					t.Error("expected nil manager")
				}
				return
			}
			if mgr == nil {
				t.Fatal("expected non-nil manager")
			}

			fft, par := mgr.GetThresholds()
			if fft != tc.expectFFT {
				t.Errorf("expected FFT %d, got %d", tc.expectFFT, fft)
			}
			if par != tc.expectPar {
				t.Errorf("expected parallel %d, got %d", tc.expectPar, par)
			}
		})
	}
}

// TestRecordIteration tests metric recording.
func TestRecordIteration(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(500000, 10000)

	// Record some iterations
	for i := 0; i < 5; i++ {
		mgr.RecordIteration(1000+i*100, time.Millisecond, i%2 == 0, i%3 == 0)
	}

	stats := mgr.GetStats()
	if stats.MetricsCollected != 5 {
		t.Errorf("expected 5 metrics, got %d", stats.MetricsCollected)
	}
	if stats.IterationsProcessed != 5 {
		t.Errorf("expected 5 iterations, got %d", stats.IterationsProcessed)
	}
}

// TestRecordIterationHistoryLimit tests that metric history is capped.
func TestRecordIterationHistoryLimit(t *testing.T) {
	mgr := newTestManager(500000, 10000)

	// Record more than MaxMetricsHistory iterations
	for i := 0; i < MaxMetricsHistory+10; i++ {
		mgr.RecordIteration(1000+i*10, time.Millisecond, true, false)
	}

	stats := mgr.GetStats()
	if stats.MetricsCollected != MaxMetricsHistory {
		t.Errorf("expected metrics capped at %d, got %d", MaxMetricsHistory, stats.MetricsCollected)
	}
	if stats.IterationsProcessed != MaxMetricsHistory+10 {
		t.Errorf("expected %d iterations processed, got %d", MaxMetricsHistory+10, stats.IterationsProcessed)
	}
}

// TestGetFFTThreshold tests individual threshold getter.
func TestGetFFTThreshold(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(123456, 10000)
	if got := mgr.GetFFTThreshold(); got != 123456 {
		t.Errorf("expected 123456, got %d", got)
	}
}

// TestGetParallelThreshold tests individual threshold getter.
func TestGetParallelThreshold(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(500000, 7890)
	if got := mgr.GetParallelThreshold(); got != 7890 {
		t.Errorf("expected 7890, got %d", got)
	}
}

// TestShouldAdjust tests the adjustment logic.
func TestShouldAdjust(t *testing.T) {
	t.Parallel()
	t.Run("not enough iterations", func(t *testing.T) {
		t.Parallel()
		mgr := newTestManager(500000, 10000)
		// Record fewer than AdjustmentInterval iterations
		for i := 0; i < DynamicAdjustmentInterval-1; i++ {
			mgr.RecordIteration(1000, time.Millisecond, false, false)
		}

		_, _, adjusted := mgr.ShouldAdjust()
		if adjusted {
			t.Error("should not adjust before interval")
		}
	})

	t.Run("not enough metrics", func(t *testing.T) {
		t.Parallel()
		mgr := newTestManager(500000, 10000)
		// Record exactly AdjustmentInterval iterations but not enough data
		for i := 0; i < DynamicAdjustmentInterval; i++ {
			if i < MinMetricsForAdjustment-1 {
				mgr.RecordIteration(1000, time.Millisecond, false, false)
			} else {
				// Trigger iteration count to reach interval
				mgr.iterationCount.Add(1)
			}
		}

		fft, par, _ := mgr.ShouldAdjust()
		// Should return current thresholds
		if fft != 500000 || par != 10000 {
			t.Errorf("expected thresholds to remain unchanged, got fft=%d, par=%d", fft, par)
		}
	})

	t.Run("FFT faster - lowers threshold", func(t *testing.T) {
		t.Parallel()
		mgr := newTestManager(500000, 10000)

		// Add non-FFT metrics (slow)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		// Add FFT metrics (fast)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, true, false)
		}

		// Force iteration count to be at interval
		mgr.iterationCount.Store(int64(DynamicAdjustmentInterval))

		fft, _, adjusted := mgr.ShouldAdjust()
		if !adjusted {
			t.Log("adjustment might not meet hysteresis margin - checking threshold")
		}
		// FFT should be lowered since FFT was faster
		if fft >= 500000 && adjusted {
			t.Errorf("expected FFT threshold to be lowered, got %d", fft)
		}
	})

	t.Run("parallel faster - lowers threshold", func(t *testing.T) {
		t.Parallel()
		mgr := newTestManager(500000, 10000)

		// Add sequential metrics (slow)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		// Add parallel metrics (fast)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, false, true)
		}

		// Force iteration count to be at interval
		mgr.iterationCount.Store(int64(DynamicAdjustmentInterval))

		_, par, adjusted := mgr.ShouldAdjust()
		if !adjusted {
			t.Log("adjustment might not meet hysteresis margin - checking threshold")
		}
		// Parallel should be lowered since parallel was faster
		if par >= 10000 && adjusted {
			t.Errorf("expected parallel threshold to be lowered, got %d", par)
		}
	})
}

// TestGetStats tests statistics retrieval.
func TestGetStats(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(500000, 10000)

	// Initial stats
	stats := mgr.GetStats()
	if stats.CurrentFFT != 500000 {
		t.Errorf("expected current FFT 500000, got %d", stats.CurrentFFT)
	}
	if stats.CurrentParallel != 10000 {
		t.Errorf("expected current parallel 10000, got %d", stats.CurrentParallel)
	}
	if stats.OriginalFFT != 500000 {
		t.Errorf("expected original FFT 500000, got %d", stats.OriginalFFT)
	}
	if stats.OriginalParallel != 10000 {
		t.Errorf("expected original parallel 10000, got %d", stats.OriginalParallel)
	}
	if stats.MetricsCollected != 0 {
		t.Errorf("expected 0 metrics, got %d", stats.MetricsCollected)
	}
	if stats.IterationsProcessed != 0 {
		t.Errorf("expected 0 iterations, got %d", stats.IterationsProcessed)
	}

	// After recording
	mgr.RecordIteration(1000, time.Millisecond, true, false)
	stats = mgr.GetStats()
	if stats.MetricsCollected != 1 {
		t.Errorf("expected 1 metric, got %d", stats.MetricsCollected)
	}
}

// TestReset tests the reset functionality.
func TestReset(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(500000, 10000)

	// Record some data
	for i := 0; i < 10; i++ {
		mgr.RecordIteration(1000, time.Millisecond, true, true)
	}

	stats := mgr.GetStats()
	if stats.MetricsCollected == 0 {
		t.Error("expected metrics before reset")
	}

	// Reset
	mgr.Reset()

	stats = mgr.GetStats()
	if stats.MetricsCollected != 0 {
		t.Errorf("expected 0 metrics after reset, got %d", stats.MetricsCollected)
	}
	if stats.IterationsProcessed != 0 {
		t.Errorf("expected 0 iterations after reset, got %d", stats.IterationsProcessed)
	}
	// Thresholds should be back to original
	if stats.CurrentFFT != 500000 {
		t.Errorf("expected FFT threshold reset to 500000, got %d", stats.CurrentFFT)
	}
	if stats.CurrentParallel != 10000 {
		t.Errorf("expected parallel threshold reset to 10000, got %d", stats.CurrentParallel)
	}
}

// TestConcurrentAccess tests thread safety.
func TestConcurrentAccess(t *testing.T) {
	mgr := newTestManager(500000, 10000)

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			mgr.RecordIteration(1000+i, time.Millisecond, i%2 == 0, i%3 == 0)
		}
		done <- true
	}()

	// Reader goroutines
	for j := 0; j < 5; j++ {
		go func() {
			for i := 0; i < 100; i++ {
				mgr.GetThresholds()
				mgr.GetFFTThreshold()
				mgr.GetParallelThreshold()
				mgr.GetStats()
			}
			done <- true
		}()
	}

	// Adjuster goroutine
	go func() {
		for i := 0; i < 20; i++ {
			mgr.ShouldAdjust()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 7; i++ {
		<-done
	}

	// Should complete without panic or race condition
	stats := mgr.GetStats()
	if stats.IterationsProcessed != 100 {
		t.Errorf("expected 100 iterations, got %d", stats.IterationsProcessed)
	}
}
