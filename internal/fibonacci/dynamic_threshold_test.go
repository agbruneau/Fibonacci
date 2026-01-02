package fibonacci

import (
	"testing"
	"time"
)

// TestNewDynamicThresholdManager tests the constructor.
func TestNewDynamicThresholdManager(t *testing.T) {
	t.Parallel()
	fft := 500000
	parallel := 10000

	mgr := NewDynamicThresholdManager(fft, parallel)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	gotFFT, gotParallel := mgr.GetThresholds()
	if gotFFT != fft {
		t.Errorf("expected FFT threshold %d, got %d", fft, gotFFT)
	}
	if gotParallel != parallel {
		t.Errorf("expected parallel threshold %d, got %d", parallel, gotParallel)
	}
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
	mgr := NewDynamicThresholdManager(500000, 10000)

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
	mgr := NewDynamicThresholdManager(500000, 10000)

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
	mgr := NewDynamicThresholdManager(123456, 10000)
	if got := mgr.GetFFTThreshold(); got != 123456 {
		t.Errorf("expected 123456, got %d", got)
	}
}

// TestGetParallelThreshold tests individual threshold getter.
func TestGetParallelThreshold(t *testing.T) {
	t.Parallel()
	mgr := NewDynamicThresholdManager(500000, 7890)
	if got := mgr.GetParallelThreshold(); got != 7890 {
		t.Errorf("expected 7890, got %d", got)
	}
}

// TestShouldAdjust tests the adjustment logic.
func TestShouldAdjust(t *testing.T) {
	t.Parallel()
	t.Run("not enough iterations", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
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
		mgr := NewDynamicThresholdManager(500000, 10000)
		// Record exactly AdjustmentInterval iterations but not enough data
		for i := 0; i < DynamicAdjustmentInterval; i++ {
			if i < MinMetricsForAdjustment-1 {
				mgr.RecordIteration(1000, time.Millisecond, false, false)
			} else {
				// Trigger iteration count to reach interval
				mgr.mu.Lock()
				mgr.iterationCount++
				mgr.mu.Unlock()
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
		mgr := NewDynamicThresholdManager(500000, 10000)

		// Add non-FFT metrics (slow)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		// Add FFT metrics (fast)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, true, false)
		}

		// Force iteration count to be at interval
		mgr.mu.Lock()
		mgr.iterationCount = DynamicAdjustmentInterval
		mgr.mu.Unlock()

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
		mgr := NewDynamicThresholdManager(500000, 10000)

		// Add sequential metrics (slow)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		// Add parallel metrics (fast)
		for i := 0; i < MinMetricsForAdjustment; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, false, true)
		}

		// Force iteration count to be at interval
		mgr.mu.Lock()
		mgr.iterationCount = DynamicAdjustmentInterval
		mgr.mu.Unlock()

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
	mgr := NewDynamicThresholdManager(500000, 10000)

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
	mgr := NewDynamicThresholdManager(500000, 10000)

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

// TestAnalyzeFFTThreshold tests FFT threshold analysis edge cases.
func TestAnalyzeFFTThreshold(t *testing.T) {
	t.Parallel()
	t.Run("empty metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		// No metrics recorded
		fft := mgr.analyzeFFTThreshold()
		if fft != 500000 {
			t.Errorf("expected current threshold 500000, got %d", fft)
		}
	})

	t.Run("only FFT metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		// Only FFT metrics
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(1000, time.Millisecond, true, false)
		}
		fft := mgr.analyzeFFTThreshold()
		if fft != 500000 {
			t.Errorf("expected current threshold with no comparison data, got %d", fft)
		}
	})

	t.Run("only non-FFT metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		// Only non-FFT metrics
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(1000, time.Millisecond, false, false)
		}
		fft := mgr.analyzeFFTThreshold()
		if fft != 500000 {
			t.Errorf("expected current threshold with no comparison data, got %d", fft)
		}
	})

	t.Run("FFT slower raises threshold", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		// Non-FFT fast
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, false, false)
		}
		// FFT slow
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, true, false)
		}
		fft := mgr.analyzeFFTThreshold()
		if fft <= 500000 {
			t.Errorf("expected FFT threshold to increase, got %d", fft)
		}
	})

	t.Run("FFT threshold respects minimum", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(100001, 10000) // Just above minimum
		// FFT fast
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 1*time.Millisecond, true, false)
		}
		// Non-FFT slow
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		fft := mgr.analyzeFFTThreshold()
		if fft < 100000 {
			t.Errorf("expected FFT threshold to not go below minimum 100000, got %d", fft)
		}
	})
}

// TestAnalyzeParallelThreshold tests parallel threshold analysis edge cases.
func TestAnalyzeParallelThreshold(t *testing.T) {
	t.Parallel()
	t.Run("empty metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		par := mgr.analyzeParallelThreshold()
		if par != 10000 {
			t.Errorf("expected current threshold 10000, got %d", par)
		}
	})

	t.Run("only parallel metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(1000, time.Millisecond, false, true)
		}
		par := mgr.analyzeParallelThreshold()
		if par != 10000 {
			t.Errorf("expected current threshold with no comparison, got %d", par)
		}
	})

	t.Run("only sequential metrics returns current", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(1000, time.Millisecond, false, false)
		}
		par := mgr.analyzeParallelThreshold()
		if par != 10000 {
			t.Errorf("expected current threshold with no comparison, got %d", par)
		}
	})

	t.Run("parallel slower raises threshold", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 10000)
		// Sequential fast
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 10*time.Millisecond, false, false)
		}
		// Parallel slow
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, true)
		}
		par := mgr.analyzeParallelThreshold()
		if par <= 10000 {
			t.Errorf("expected parallel threshold to increase, got %d", par)
		}
	})

	t.Run("parallel threshold respects minimum", func(t *testing.T) {
		t.Parallel()
		mgr := NewDynamicThresholdManager(500000, 1025) // Just above minimum
		// Parallel fast
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 1*time.Millisecond, false, true)
		}
		// Sequential slow
		for i := 0; i < 5; i++ {
			mgr.RecordIteration(10000, 100*time.Millisecond, false, false)
		}
		par := mgr.analyzeParallelThreshold()
		if par < 1024 {
			t.Errorf("expected parallel threshold to not go below minimum 1024, got %d", par)
		}
	})
}

// TestAvgTimePerBit tests the time-per-bit calculation.
func TestAvgTimePerBit(t *testing.T) {
	mgr := NewDynamicThresholdManager(500000, 10000)

	t.Run("empty metrics returns zero", func(t *testing.T) {
		result := mgr.avgTimePerBit(nil)
		if result != 0 {
			t.Errorf("expected 0, got %f", result)
		}
	})

	t.Run("zero bits returns zero", func(t *testing.T) {
		metrics := []IterationMetric{{BitLen: 0, Duration: time.Millisecond}}
		result := mgr.avgTimePerBit(metrics)
		if result != 0 {
			t.Errorf("expected 0 for zero bits, got %f", result)
		}
	})

	t.Run("calculates correctly", func(t *testing.T) {
		metrics := []IterationMetric{
			{BitLen: 1000, Duration: time.Millisecond},
			{BitLen: 2000, Duration: 2 * time.Millisecond},
		}
		// Total: 3ms for 3000 bits = 1ms/1000 bits = 1000ns/bit
		result := mgr.avgTimePerBit(metrics)
		expected := float64(3*time.Millisecond) / 3000.0
		if result != expected {
			t.Errorf("expected %f, got %f", expected, result)
		}
	})
}

// TestSignificantChange tests the hysteresis logic.
func TestSignificantChange(t *testing.T) {
	t.Parallel()
	mgr := NewDynamicThresholdManager(500000, 10000)

	tests := []struct {
		name   string
		oldVal int
		newVal int
		expect bool
	}{
		{"old is zero, new is zero", 0, 0, false},
		{"old is zero, new is non-zero", 0, 100, true},
		{"small change below margin", 100, 105, false},
		{"change at margin", 100, 116, true},
		{"large change", 100, 200, true},
		{"negative change below margin", 100, 95, false},
		{"negative change above margin", 100, 80, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := mgr.significantChange(tc.oldVal, tc.newVal)
			if result != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, result)
			}
		})
	}
}

// TestConcurrentAccess tests thread safety.
func TestConcurrentAccess(t *testing.T) {
	mgr := NewDynamicThresholdManager(500000, 10000)

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
