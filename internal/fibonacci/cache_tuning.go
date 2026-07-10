package fibonacci

import (
	"github.com/agbruneau/FibGo/internal/bigfft"
)

// Cache tuning: ExecuteDoublingLoop (doubling_framework.go) samples the
// global bigfft FFT transform cache every cacheSampleInterval iterations
// (and always on the final iteration) while dynamic thresholds are enabled,
// and applies the P1-02 grow/shrink heuristic below. This used to be
// dispatched through a CacheStrategy interface with one production
// implementation and one test mock — pure indirection for a single call
// site, removed in favor of calling decideCacheTuning directly (PLAN.md
// wave-1 cut-list, tag `yagni`).
const (
	// cacheSampleInterval matches the original P1-02 throttle (every 8 iters).
	cacheSampleInterval = 8

	// cacheMaxEntriesUpperBound prevents unbounded growth of the LRU cache.
	cacheMaxEntriesUpperBound = 8192

	// cacheGrowthFactor / cacheShrinkFactor mirror the original 20% grow / 10%
	// shrink heuristics.
	cacheGrowthFactor   = 1.2
	cacheMinBitLenGrow  = 1.1
	cacheHitRateGrow    = 0.5
	cacheHitRateShrink  = 0.1
	cacheActivityWindow = 10
)

// decideCacheTuning applies the P1-02 grow/shrink heuristics to a
// stats/config snapshot and returns the adjusted config; changed is false
// when no adjustment is warranted. Pure function so the rules are unit-
// testable without touching the global bigfft cache — see
// TestDecideCacheTuning in cache_tuning_test.go.
func decideCacheTuning(stats bigfft.CacheStats, cfg bigfft.TransformCacheConfig) (bigfft.TransformCacheConfig, bool) {
	// If evicting frequently with good hit rate, increase cache size.
	if stats.Evictions > 0 && stats.HitRate > cacheHitRateGrow {
		if cfg.MaxEntries < cacheMaxEntriesUpperBound {
			cfg.MaxEntries = int(float64(cfg.MaxEntries) * cacheGrowthFactor)
			if cfg.MaxEntries > cacheMaxEntriesUpperBound {
				cfg.MaxEntries = cacheMaxEntriesUpperBound
			}
			return cfg, true
		}
		return cfg, false
	}

	// If cache is not useful, prune smaller transforms.
	if stats.HitRate < cacheHitRateShrink && (stats.Misses+stats.Hits) > cacheActivityWindow {
		cfg.MinBitLen = int(float64(cfg.MinBitLen) * cacheMinBitLenGrow)
		return cfg, true
	}

	return cfg, false
}
