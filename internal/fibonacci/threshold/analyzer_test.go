package threshold

import (
	"testing"
	"time"
)

func TestThresholdAnalyzer_CalculateSpeedupRatio(t *testing.T) {
	t.Parallel()
	a := ThresholdAnalyzer{}

	tests := []struct {
		name        string
		optimized   float64
		baseline    float64
		expected    float64
		expectExact bool
	}{
		{"normal speedup", 1.0, 2.0, 2.0, true},
		{"slowdown", 2.0, 1.0, 0.5, true},
		{"equal", 1.0, 1.0, 1.0, true},
		{"zero optimized", 0, 1.0, 0, true},
		{"zero baseline", 1.0, 0, 0, true},
		{"negative optimized", -1.0, 1.0, 0, true},
		{"negative baseline", 1.0, -1.0, 0, true},
		{"both zero", 0, 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := a.CalculateSpeedupRatio(tc.optimized, tc.baseline)
			if got != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, got)
			}
		})
	}
}

func TestThresholdAnalyzer_AvgTimePerBit(t *testing.T) {
	t.Parallel()
	a := ThresholdAnalyzer{}

	t.Run("nil metrics", func(t *testing.T) {
		if got := a.AvgTimePerBit(nil); got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})

	t.Run("zero total bits", func(t *testing.T) {
		metrics := []IterationMetric{{BitLen: 0, Duration: time.Millisecond}}
		if got := a.AvgTimePerBit(metrics); got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})

	t.Run("computes average", func(t *testing.T) {
		metrics := []IterationMetric{
			{BitLen: 1000, Duration: time.Millisecond},
			{BitLen: 2000, Duration: 2 * time.Millisecond},
		}
		expected := float64(3*time.Millisecond) / 3000.0
		if got := a.AvgTimePerBit(metrics); got != expected {
			t.Errorf("expected %f, got %f", expected, got)
		}
	})
}

func TestThresholdAnalyzer_SignificantChange(t *testing.T) {
	t.Parallel()
	a := ThresholdAnalyzer{}

	tests := []struct {
		name   string
		oldVal int
		newVal int
		expect bool
	}{
		{"both zero", 0, 0, false},
		{"old zero, new non-zero", 0, 100, true},
		{"small positive change below margin", 100, 105, false},
		{"positive change above margin", 100, 116, true},
		{"large positive change", 100, 200, true},
		{"small negative change below margin", 100, 95, false},
		{"negative change above margin", 100, 80, true},
		{"identical", 500, 500, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := a.SignificantChange(tc.oldVal, tc.newVal); got != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}

func TestThresholdAnalyzer_ApplyThresholdAdjustment(t *testing.T) {
	t.Parallel()
	a := ThresholdAnalyzer{}
	baseParams := AnalysisParams{
		SpeedupThreshold:  1.2,
		LowerNumerator:    9,
		RaiseNumerator:    11,
		MinThreshold:      100,
		MaxCapMultiplier:  2,
		CurrentThreshold:  1000,
		OriginalThreshold: 1000,
	}

	t.Run("ratio in dead band keeps current", func(t *testing.T) {
		got := a.ApplyThresholdAdjustment(1.0, baseParams)
		if got != 1000 {
			t.Errorf("expected 1000, got %d", got)
		}
	})

	t.Run("optimized faster lowers threshold", func(t *testing.T) {
		got := a.ApplyThresholdAdjustment(2.0, baseParams)
		// 1000 * 9 / 10 = 900
		if got != 900 {
			t.Errorf("expected 900, got %d", got)
		}
	})

	t.Run("optimized slower raises threshold", func(t *testing.T) {
		got := a.ApplyThresholdAdjustment(0.5, baseParams)
		// 1000 * 11 / 10 = 1100
		if got != 1100 {
			t.Errorf("expected 1100, got %d", got)
		}
	})

	t.Run("lowering respects MinThreshold floor", func(t *testing.T) {
		params := baseParams
		params.CurrentThreshold = 105
		params.MinThreshold = 100
		// 105 * 9 / 10 = 94 (int truncation), clamped to 100.
		got := a.ApplyThresholdAdjustment(2.0, params)
		if got != 100 {
			t.Errorf("expected floor 100, got %d", got)
		}
	})

	t.Run("raising respects MaxCapMultiplier ceiling", func(t *testing.T) {
		params := baseParams
		params.CurrentThreshold = 1900
		params.OriginalThreshold = 1000
		params.MaxCapMultiplier = 2
		// 1900 * 11 / 10 = 2090, clamped to 2000.
		got := a.ApplyThresholdAdjustment(0.5, params)
		if got != 2000 {
			t.Errorf("expected cap 2000, got %d", got)
		}
	})
}

func TestThresholdAnalyzer_Analyze(t *testing.T) {
	t.Parallel()
	a := ThresholdAnalyzer{}

	predicateFFT := func(m IterationMetric) bool { return m.UsedFFT }
	baseParams := AnalysisParams{
		Predicate:         predicateFFT,
		SpeedupThreshold:  FFTSpeedupThreshold,
		LowerNumerator:    9,
		RaiseNumerator:    11,
		MinThreshold:      100000,
		MaxCapMultiplier:  2,
		CurrentThreshold:  500000,
		OriginalThreshold: 500000,
	}

	t.Run("empty metrics keeps current", func(t *testing.T) {
		if got := a.Analyze(nil, baseParams); got != 500000 {
			t.Errorf("expected 500000, got %d", got)
		}
	})

	t.Run("only matching metrics keeps current", func(t *testing.T) {
		metrics := []IterationMetric{
			{BitLen: 1000, Duration: time.Millisecond, UsedFFT: true},
			{BitLen: 1000, Duration: time.Millisecond, UsedFFT: true},
		}
		if got := a.Analyze(metrics, baseParams); got != 500000 {
			t.Errorf("expected 500000, got %d", got)
		}
	})

	t.Run("only non-matching metrics keeps current", func(t *testing.T) {
		metrics := []IterationMetric{
			{BitLen: 1000, Duration: time.Millisecond, UsedFFT: false},
			{BitLen: 1000, Duration: time.Millisecond, UsedFFT: false},
		}
		if got := a.Analyze(metrics, baseParams); got != 500000 {
			t.Errorf("expected 500000, got %d", got)
		}
	})

	t.Run("FFT faster lowers threshold", func(t *testing.T) {
		metrics := []IterationMetric{
			// Non-FFT samples — slow.
			{BitLen: 10000, Duration: 100 * time.Millisecond, UsedFFT: false},
			{BitLen: 10000, Duration: 100 * time.Millisecond, UsedFFT: false},
			// FFT samples — fast.
			{BitLen: 10000, Duration: 10 * time.Millisecond, UsedFFT: true},
			{BitLen: 10000, Duration: 10 * time.Millisecond, UsedFFT: true},
		}
		got := a.Analyze(metrics, baseParams)
		if got >= 500000 {
			t.Errorf("expected lowered threshold, got %d", got)
		}
		if got < 100000 {
			t.Errorf("expected threshold ≥ 100000 floor, got %d", got)
		}
	})

	t.Run("FFT slower raises threshold", func(t *testing.T) {
		metrics := []IterationMetric{
			// Non-FFT samples — fast.
			{BitLen: 10000, Duration: 10 * time.Millisecond, UsedFFT: false},
			{BitLen: 10000, Duration: 10 * time.Millisecond, UsedFFT: false},
			// FFT samples — slow.
			{BitLen: 10000, Duration: 100 * time.Millisecond, UsedFFT: true},
			{BitLen: 10000, Duration: 100 * time.Millisecond, UsedFFT: true},
		}
		got := a.Analyze(metrics, baseParams)
		if got <= 500000 {
			t.Errorf("expected raised threshold, got %d", got)
		}
	})

	t.Run("equal speed within dead band keeps current", func(t *testing.T) {
		metrics := []IterationMetric{
			{BitLen: 10000, Duration: 50 * time.Millisecond, UsedFFT: false},
			{BitLen: 10000, Duration: 50 * time.Millisecond, UsedFFT: false},
			{BitLen: 10000, Duration: 50 * time.Millisecond, UsedFFT: true},
			{BitLen: 10000, Duration: 50 * time.Millisecond, UsedFFT: true},
		}
		if got := a.Analyze(metrics, baseParams); got != 500000 {
			t.Errorf("expected 500000 (in dead band), got %d", got)
		}
	})
}

func TestFilterMetricsByMode(t *testing.T) {
	t.Parallel()
	metrics := []IterationMetric{
		{BitLen: 1, UsedFFT: true},
		{BitLen: 2, UsedFFT: false},
		{BitLen: 3, UsedFFT: true},
		{BitLen: 4, UsedFFT: false},
	}
	matching, nonMatching := filterMetricsByMode(metrics, func(m IterationMetric) bool { return m.UsedFFT })
	if len(matching) != 2 {
		t.Errorf("matching: expected 2, got %d", len(matching))
	}
	if len(nonMatching) != 2 {
		t.Errorf("nonMatching: expected 2, got %d", len(nonMatching))
	}
	for _, m := range matching {
		if !m.UsedFFT {
			t.Errorf("matching contained non-FFT metric: %+v", m)
		}
	}
	for _, m := range nonMatching {
		if m.UsedFFT {
			t.Errorf("nonMatching contained FFT metric: %+v", m)
		}
	}
}
