package fibonacci

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/bigfft"
)

// TestDecideCacheTuning unit-tests the P1-02 grow/shrink heuristics in
// isolation. Before this table existed the rules were only exercised
// indirectly through full FFT calculations (cache_bench_test.go), so a
// broken threshold or factor would not have failed any unit test.
func TestDecideCacheTuning(t *testing.T) {
	t.Parallel()

	baseCfg := bigfft.TransformCacheConfig{
		MaxEntries: 256,
		MinBitLen:  100000,
		Enabled:    true,
	}

	cases := []struct {
		name           string
		stats          bigfft.CacheStats
		cfg            bigfft.TransformCacheConfig
		wantChanged    bool
		wantMaxEntries int
		wantMinBitLen  int
	}{
		{
			name:           "grows MaxEntries on evictions with good hit rate",
			stats:          bigfft.CacheStats{Evictions: 5, HitRate: 0.6},
			cfg:            baseCfg,
			wantChanged:    true,
			wantMaxEntries: 307, // 256 * 1.2
			wantMinBitLen:  100000,
		},
		{
			name:           "does not grow past the upper bound",
			stats:          bigfft.CacheStats{Evictions: 5, HitRate: 0.9},
			cfg:            bigfft.TransformCacheConfig{MaxEntries: cacheMaxEntriesUpperBound, MinBitLen: 100000, Enabled: true},
			wantChanged:    false,
			wantMaxEntries: cacheMaxEntriesUpperBound,
			wantMinBitLen:  100000,
		},
		{
			name:           "clamps growth that would overshoot the upper bound",
			stats:          bigfft.CacheStats{Evictions: 5, HitRate: 0.9},
			cfg:            bigfft.TransformCacheConfig{MaxEntries: 8000, MinBitLen: 100000, Enabled: true},
			wantChanged:    true,
			wantMaxEntries: cacheMaxEntriesUpperBound, // 8000 * 1.2 = 9600, must clamp to 8192
			wantMinBitLen:  100000,
		},
		{
			name:           "raises MinBitLen when cache is not useful",
			stats:          bigfft.CacheStats{Hits: 1, Misses: 19, HitRate: 0.05},
			cfg:            baseCfg,
			wantChanged:    true,
			wantMaxEntries: 256,
			wantMinBitLen:  110000, // 100000 * 1.1
		},
		{
			name:           "low hit rate alone is not enough below the activity window",
			stats:          bigfft.CacheStats{Hits: 0, Misses: 5, HitRate: 0.0},
			cfg:            baseCfg,
			wantChanged:    false,
			wantMaxEntries: 256,
			wantMinBitLen:  100000,
		},
		{
			name:           "steady state is a no-op",
			stats:          bigfft.CacheStats{Hits: 50, Misses: 50, HitRate: 0.5},
			cfg:            baseCfg,
			wantChanged:    false,
			wantMaxEntries: 256,
			wantMinBitLen:  100000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, changed := decideCacheTuning(tc.stats, tc.cfg)
			if changed != tc.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if got.MaxEntries != tc.wantMaxEntries {
				t.Errorf("MaxEntries = %d, want %d", got.MaxEntries, tc.wantMaxEntries)
			}
			if got.MinBitLen != tc.wantMinBitLen {
				t.Errorf("MinBitLen = %d, want %d", got.MinBitLen, tc.wantMinBitLen)
			}
		})
	}
}

// TestDecideCacheTuningBounds pins audit L-08. The heuristics were unreachable
// until --dynamic-thresholds existed (M-04), and both growth rules had a
// degenerate case: multiplicative growth that does not move, reported as a
// change, makes the caller take the cache's exclusive lock every sample to
// write back the value it already held.
func TestDecideCacheTuningBounds(t *testing.T) {
	t.Parallel()

	// Grow branch: evictions with a good hit rate.
	growStats := bigfft.CacheStats{Evictions: 1, HitRate: 0.9, Hits: 90, Misses: 10}

	t.Run("small MaxEntries still grows", func(t *testing.T) {
		t.Parallel()
		for _, start := range []int{1, 2, 3, 4} {
			got, changed := decideCacheTuning(growStats, bigfft.TransformCacheConfig{MaxEntries: start, MinBitLen: 100000})
			if !changed {
				t.Errorf("MaxEntries=%d: expected an adjustment", start)
				continue
			}
			if got.MaxEntries <= start {
				t.Errorf("MaxEntries=%d: grew to %d, which is not progress", start, got.MaxEntries)
			}
		}
	})

	t.Run("MaxEntries stops at the ceiling", func(t *testing.T) {
		t.Parallel()
		got, changed := decideCacheTuning(growStats, bigfft.TransformCacheConfig{MaxEntries: cacheMaxEntriesUpperBound, MinBitLen: 100000})
		if changed {
			t.Error("at the ceiling there is nothing left to change")
		}
		if got.MaxEntries != cacheMaxEntriesUpperBound {
			t.Errorf("MaxEntries = %d, want the ceiling %d", got.MaxEntries, cacheMaxEntriesUpperBound)
		}
	})

	// Shrink branch: a hit rate too low to be worth the work, with enough
	// activity for the reading to mean something.
	shrinkStats := bigfft.CacheStats{HitRate: 0.01, Hits: 1, Misses: 99}

	t.Run("MinBitLen grows then stops at the ceiling", func(t *testing.T) {
		t.Parallel()
		got, changed := decideCacheTuning(shrinkStats, bigfft.TransformCacheConfig{MaxEntries: 256, MinBitLen: 100000})
		if !changed || got.MinBitLen <= 100000 {
			t.Errorf("expected MinBitLen to grow from 100000, got %d (changed=%v)", got.MinBitLen, changed)
		}

		got, changed = decideCacheTuning(shrinkStats, bigfft.TransformCacheConfig{MaxEntries: 256, MinBitLen: cacheMinBitLenUpperBound})
		if changed {
			t.Error("at the ceiling there is nothing left to change")
		}
		if got.MinBitLen != cacheMinBitLenUpperBound {
			t.Errorf("MinBitLen = %d, want the ceiling %d", got.MinBitLen, cacheMinBitLenUpperBound)
		}
	})

	t.Run("no signal means no change", func(t *testing.T) {
		t.Parallel()
		cfg := bigfft.TransformCacheConfig{MaxEntries: 256, MinBitLen: 100000}
		got, changed := decideCacheTuning(bigfft.CacheStats{HitRate: 0.3, Hits: 30, Misses: 70}, cfg)
		if changed || got != cfg {
			t.Errorf("a middling hit rate with no evictions must not adjust anything, got %+v (changed=%v)", got, changed)
		}
	})
}
