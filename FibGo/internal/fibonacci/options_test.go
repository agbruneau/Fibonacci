package fibonacci

import (
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// normalizeOptions Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestNormalizeOptions tests that default values are applied correctly.
func TestNormalizeOptions(t *testing.T) {
	t.Parallel()

	t.Run("applies all defaults when options are zero", func(t *testing.T) {
		t.Parallel()
		opts := Options{}
		normalized := normalizeOptions(opts)

		if normalized.ParallelThreshold != DefaultParallelThreshold {
			t.Errorf("ParallelThreshold = %d, want %d", normalized.ParallelThreshold, DefaultParallelThreshold)
		}
		if normalized.FFTThreshold != DefaultFFTThreshold {
			t.Errorf("FFTThreshold = %d, want %d", normalized.FFTThreshold, DefaultFFTThreshold)
		}
		if normalized.StrassenThreshold != DefaultStrassenThreshold {
			t.Errorf("StrassenThreshold = %d, want %d", normalized.StrassenThreshold, DefaultStrassenThreshold)
		}
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		t.Parallel()
		opts := Options{
			ParallelThreshold: 1234,
			FFTThreshold:      5678,
			StrassenThreshold: 3456,
		}
		normalized := normalizeOptions(opts)

		if normalized.ParallelThreshold != 1234 {
			t.Errorf("ParallelThreshold = %d, want 1234", normalized.ParallelThreshold)
		}
		if normalized.FFTThreshold != 5678 {
			t.Errorf("FFTThreshold = %d, want 5678", normalized.FFTThreshold)
		}
		if normalized.StrassenThreshold != 3456 {
			t.Errorf("StrassenThreshold = %d, want 3456", normalized.StrassenThreshold)
		}
	})

	t.Run("applies defaults only to zero values", func(t *testing.T) {
		t.Parallel()
		opts := Options{
			ParallelThreshold: 9999,
			FFTThreshold:      0, // This should get default
		}
		normalized := normalizeOptions(opts)

		if normalized.ParallelThreshold != 9999 {
			t.Errorf("ParallelThreshold = %d, want 9999", normalized.ParallelThreshold)
		}
		if normalized.FFTThreshold != DefaultFFTThreshold {
			t.Errorf("FFTThreshold = %d, want %d", normalized.FFTThreshold, DefaultFFTThreshold)
		}
	})

	t.Run("does not modify original options", func(t *testing.T) {
		t.Parallel()
		original := Options{
			ParallelThreshold: 0,
			FFTThreshold:      0,
		}
		_ = normalizeOptions(original)

		// Original should remain unchanged
		if original.ParallelThreshold != 0 {
			t.Errorf("original.ParallelThreshold was modified to %d", original.ParallelThreshold)
		}
		if original.FFTThreshold != 0 {
			t.Errorf("original.FFTThreshold was modified to %d", original.FFTThreshold)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Options Struct Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestOptionsFFTCacheEnabled tests the pointer semantics of FFTCacheEnabled.
func TestOptionsFFTCacheEnabled(t *testing.T) {
	t.Parallel()

	t.Run("nil means default enabled", func(t *testing.T) {
		t.Parallel()
		opts := Options{}
		if opts.FFTCacheEnabled != nil {
			t.Error("expected FFTCacheEnabled to be nil by default")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		t.Parallel()
		enabled := true
		opts := Options{FFTCacheEnabled: &enabled}
		if opts.FFTCacheEnabled == nil || !*opts.FFTCacheEnabled {
			t.Error("expected FFTCacheEnabled to be true")
		}
	})

	t.Run("explicit false", func(t *testing.T) {
		t.Parallel()
		disabled := false
		opts := Options{FFTCacheEnabled: &disabled}
		if opts.FFTCacheEnabled == nil || *opts.FFTCacheEnabled {
			t.Error("expected FFTCacheEnabled to be false")
		}
	})
}
