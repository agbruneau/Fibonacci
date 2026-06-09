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
