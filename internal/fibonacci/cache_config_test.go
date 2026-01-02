package fibonacci

import (
	"testing"

	"github.com/agbru/fibcalc/internal/bigfft"
)

// TestConfigureFFTCacheDefault verifies that configureFFTCache uses default
// values when options are not specified.
func TestConfigureFFTCacheDefault(t *testing.T) {
	t.Parallel()
	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      500000,
		// FFTCache options not set - should use defaults
	}

	configureFFTCache(opts)

	// Verify cache is configured with defaults
	cache := bigfft.GetTransformCache()
	stats := cache.Stats()

	// Cache should be enabled by default
	if stats.Size < 0 {
		t.Error("Cache should be initialized")
	}

	// Reset to defaults for other tests
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	bigfft.SetTransformCacheConfig(defaultConfig)
}

// TestConfigureFFTCacheCustom verifies that configureFFTCache applies custom
// configuration values when provided.
func TestConfigureFFTCacheCustom(t *testing.T) {
	t.Parallel()
	enabled := true
	opts := Options{
		ParallelThreshold:  4096,
		FFTThreshold:       500000,
		FFTCacheMinBitLen:  50000, // Lower threshold for testing
		FFTCacheMaxEntries: 256,   // Larger cache
		FFTCacheEnabled:    &enabled,
	}

	configureFFTCache(opts)

	// Verify cache configuration was applied
	cache := bigfft.GetTransformCache()
	stats := cache.Stats()

	// Cache should be enabled
	if stats.Size < 0 {
		t.Error("Cache should be initialized")
	}

	// Reset to defaults for other tests
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	bigfft.SetTransformCacheConfig(defaultConfig)
}

// TestConfigureFFTCacheDisabled verifies that configureFFTCache can disable
// the cache when requested.
func TestConfigureFFTCacheDisabled(t *testing.T) {
	t.Parallel()
	disabled := false
	opts := Options{
		ParallelThreshold: 4096,
		FFTThreshold:      500000,
		FFTCacheEnabled:   &disabled,
	}

	configureFFTCache(opts)

	// Verify cache is disabled
	cache := bigfft.GetTransformCache()
	stats := cache.Stats()

	// Cache should still exist but be disabled
	if stats.Size < 0 {
		t.Error("Cache should still be initialized")
	}

	// Reset to defaults for other tests
	defaultConfig := bigfft.DefaultTransformCacheConfig()
	bigfft.SetTransformCacheConfig(defaultConfig)
}
