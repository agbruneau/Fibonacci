package bigfft

import (
	"testing"
)

func TestPreWarmPools(t *testing.T) {
	t.Parallel()
	// Test PreWarmPools does not panic and allocates pools
	// Pass uint64 directly as per signature
	PreWarmPools(1000)

	// Try with larger n
	PreWarmPools(100000)
}

func TestAcquireReleaseWordSlice(t *testing.T) {
	t.Parallel()
	// Normal size
	s := acquireWordSlice(100)
	if len(s) != 100 {
		t.Errorf("expected length 100, got %d", len(s))
	}
	releaseWordSlice(s)

	// Large size (beyond pools)
	sLarge := acquireWordSlice(20000000)
	if len(sLarge) != 20000000 {
		t.Errorf("expected length 20000000, got %d", len(sLarge))
	}
	releaseWordSlice(sLarge)

	// Nil release
	releaseWordSlice(nil)
}

func TestAcquireReleaseFermat(t *testing.T) {
	t.Parallel()
	s := acquireFermat(100)
	if len(s) != 100 {
		t.Errorf("expected length 100, got %d", len(s))
	}
	releaseFermat(s)

	sLarge := acquireFermat(3000000)
	if len(sLarge) != 3000000 {
		t.Errorf("expected length 3000000, got %d", len(sLarge))
	}
	releaseFermat(sLarge)

	releaseFermat(nil)
}

func TestAcquireReleaseNatSlice(t *testing.T) {
	t.Parallel()
	sLarge := acquireNatSlice(40000)
	if len(sLarge) != 40000 {
		t.Errorf("expected length 40000, got %d", len(sLarge))
	}
	releaseNatSlice(sLarge)

	releaseNatSlice(nil)
}

func TestAcquireReleaseFermatSlice(t *testing.T) {
	t.Parallel()
	s := acquireFermatSlice(10)
	if len(s) != 10 {
		t.Errorf("expected length 10, got %d", len(s))
	}
	releaseFermatSlice(s)

	sLarge := acquireFermatSlice(40000)
	if len(sLarge) != 40000 {
		t.Errorf("expected length 40000, got %d", len(sLarge))
	}
	releaseFermatSlice(sLarge)

	releaseFermatSlice(nil)
}

func TestFFTStatePoolExtra(t *testing.T) {
	t.Parallel()
	state := acquireFFTState(100, 4)
	if state == nil {
		t.Fatal("acquireFFTState returned nil")
	}
	if state.n != 100 || state.k != 4 {
		t.Errorf("state params mismatch: n=%d, k=%d", state.n, state.k)
	}
	if len(state.tmp) < 101 {
		t.Errorf("tmp buffer too small: %d", len(state.tmp))
	}

	// Reuse
	releaseFFTState(state)
	state2 := acquireFFTState(200, 5)
	if state2 == nil {
		t.Fatal("acquireFFTState (reuse) returned nil")
	}
	releaseFFTState(state2)
	releaseFFTState(nil)
}
