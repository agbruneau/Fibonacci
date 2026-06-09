package fibonacci

import (
	"github.com/agbruneau/FibGo/internal/bigfft"
)

// CacheStrategy abstracts the per-iteration FFT cache adjustment logic so the
// Fast Doubling loop does not couple directly to the bigfft package.
//
// Sample is invoked from inside the Fast Doubling loop. Implementations are
// expected to perform their own throttling (e.g. only act every N
// iterations) — the loop calls Sample on every iteration to keep the call
// site uniform. iter is the 1-based iteration counter; totalIters is the
// total number of iterations the loop will execute. Implementations should
// always run their adjustment on the final iteration (iter == totalIters)
// regardless of the throttling cadence so the cache state is updated before
// the loop exits.
//
// A nil error means "nothing to report"; non-nil errors are propagated up
// the call stack and abort the calculation.
type CacheStrategy interface {
	Sample(iter int, totalIters int) error
}

// bigfftCacheStrategy is the default CacheStrategy: it preserves the historical
// behavior previously inlined in ExecuteDoublingLoop — sample the global FFT
// transform cache every 8 iterations (and on the final iteration) and tune
// MaxEntries / MinBitLen heuristically based on hit rate and evictions.
//
// The constants below mirror the original P1-02 implementation exactly so the
// observable behavior is preserved.
type bigfftCacheStrategy struct{}

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

// NewBigFFTCacheStrategy returns the default CacheStrategy that wraps the
// global bigfft transform cache.
func NewBigFFTCacheStrategy() CacheStrategy {
	return bigfftCacheStrategy{}
}

// Sample inspects the bigfft global cache and adjusts MaxEntries / MinBitLen
// based on current hit-rate / eviction signals. It is a no-op unless iter is
// a multiple of cacheSampleInterval or this is the final iteration.
func (bigfftCacheStrategy) Sample(iter int, totalIters int) error {
	// Throttle: only sample every cacheSampleInterval iterations, and always
	// on the final iteration.
	if iter%cacheSampleInterval != 0 && iter != totalIters {
		return nil
	}

	cache := bigfft.GetTransformCache()
	if cfg, changed := decideCacheTuning(cache.Stats(), cache.Config()); changed {
		bigfft.SetTransformCacheConfig(cfg)
	}
	return nil
}

// decideCacheTuning applies the P1-02 grow/shrink heuristics to a
// stats/config snapshot and returns the adjusted config; changed is false
// when no adjustment is warranted. Pure function so the rules are unit-
// testable without touching the global bigfft cache (the throttling and the
// application of the result stay in Sample, whose calling discipline is
// covered by the mockCacheStrategy tests in doubling_framework_test.go).
func decideCacheTuning(stats bigfft.CacheStats, cfg bigfft.TransformCacheConfig) (bigfft.TransformCacheConfig, bool) {
	// If evicting frequently with good hit rate, increase cache size.
	if stats.Evictions > 0 && stats.HitRate > cacheHitRateGrow {
		if cfg.MaxEntries < cacheMaxEntriesUpperBound {
			cfg.MaxEntries = int(float64(cfg.MaxEntries) * cacheGrowthFactor)
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
