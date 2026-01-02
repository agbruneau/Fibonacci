package bigfft

import (
	"math/big"
	"sync"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// TransformCache Unit Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestNewTransformCache(t *testing.T) {
	t.Parallel()
	config := DefaultTransformCacheConfig()
	cache := NewTransformCache(config)

	if cache == nil {
		t.Fatal("NewTransformCache returned nil")
	}
	if cache.config.MaxEntries != 128 {
		t.Errorf("expected MaxEntries=128, got %d", cache.config.MaxEntries)
	}
	if cache.config.MinBitLen != 100000 {
		t.Errorf("expected MinBitLen=100000, got %d", cache.config.MinBitLen)
	}
	if !cache.config.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestGetTransformCache(t *testing.T) {
	cache1 := GetTransformCache()
	cache2 := GetTransformCache()

	if cache1 != cache2 {
		t.Error("GetTransformCache should return the same instance")
	}
}

func TestSetTransformCacheConfig(t *testing.T) {
	t.Parallel()

	t.Run("Set config with enabled cache", func(t *testing.T) {
		t.Parallel()
		config := TransformCacheConfig{
			MaxEntries: 64,
			MinBitLen:  50000,
			Enabled:    true,
		}

		SetTransformCacheConfig(config)

		cache := GetTransformCache()
		cache.mu.Lock()
		defer cache.mu.Unlock()

		if cache.config.MaxEntries != config.MaxEntries {
			t.Errorf("Expected MaxEntries=%d, got %d", config.MaxEntries, cache.config.MaxEntries)
		}
		if cache.config.MinBitLen != config.MinBitLen {
			t.Errorf("Expected MinBitLen=%d, got %d", config.MinBitLen, cache.config.MinBitLen)
		}
		if cache.config.Enabled != config.Enabled {
			t.Errorf("Expected Enabled=%v, got %v", config.Enabled, cache.config.Enabled)
		}
	})

	t.Run("Set config with disabled cache clears entries", func(t *testing.T) {
		t.Parallel()
		// First enable and add some entries using Put
		config1 := TransformCacheConfig{
			MaxEntries: 64,
			MinBitLen:  50000,
			Enabled:    true,
		}
		SetTransformCacheConfig(config1)

		cache := GetTransformCache()
		// Add a real entry using Put
		testData := make(nat, 100)
		testData[0] = big.Word(123)
		mockValues := PolValues{
			K:      4,
			N:      10,
			Values: make([]fermat, 16),
		}
		for i := range mockValues.Values {
			mockValues.Values[i] = make(fermat, 101)
		}
		cache.Put(testData, mockValues)

		cache.mu.Lock()
		entryCount := len(cache.entries)
		cache.mu.Unlock()

		if entryCount == 0 {
			t.Error("Expected at least one entry before disabling")
		}

		// Now disable
		config2 := TransformCacheConfig{
			MaxEntries: 64,
			MinBitLen:  50000,
			Enabled:    false,
		}
		SetTransformCacheConfig(config2)

		cache.mu.Lock()
		entryCountAfter := len(cache.entries)
		cache.mu.Unlock()

		if entryCountAfter != 0 {
			t.Errorf("Expected cache to be cleared when disabled, got %d entries", entryCountAfter)
		}
	})
}

func TestTransformCachePutAndGet(t *testing.T) {
	t.Parallel()
	// Create a cache with low threshold for testing
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  64, // Low threshold for testing
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	// Create test data
	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	// Create mock PolValues
	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}, {4, 5, 6}},
	}

	// Put into cache
	cache.Put(testData, mockValues)

	// Get from cache
	result, found := cache.Get(testData, 4, 100)
	if !found {
		t.Fatal("expected to find cached value")
	}

	if result.K != mockValues.K {
		t.Errorf("expected K=%d, got %d", mockValues.K, result.K)
	}
	if result.N != mockValues.N {
		t.Errorf("expected N=%d, got %d", mockValues.N, result.N)
	}
	if len(result.Values) != len(mockValues.Values) {
		t.Errorf("expected %d values, got %d", len(mockValues.Values), len(result.Values))
	}
}

func TestTransformCacheMiss(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	// Try to get without putting first
	_, found := cache.Get(testData, 4, 100)
	if found {
		t.Error("expected cache miss for non-existent key")
	}
}

func TestTransformCacheEviction(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 3, // Small for testing eviction
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	// Add more entries than MaxEntries
	for i := 0; i < 5; i++ {
		testData := make(nat, 10)
		testData[0] = big.Word(i) // Different data for each entry

		mockValues := PolValues{
			K:      4,
			N:      100,
			Values: []fermat{{big.Word(i)}},
		}
		cache.Put(testData, mockValues)
	}

	stats := cache.Stats()
	if stats.Size > config.MaxEntries {
		t.Errorf("cache size %d exceeds MaxEntries %d", stats.Size, config.MaxEntries)
	}
	if stats.Evictions == 0 {
		t.Error("expected evictions to occur")
	}
}

func TestTransformCacheDisabled(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  64,
		Enabled:    false, // Disabled
	}
	cache := NewTransformCache(config)

	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}},
	}

	// Put should be a no-op when disabled
	cache.Put(testData, mockValues)

	// Get should return false when disabled
	_, found := cache.Get(testData, 4, 100)
	if found {
		t.Error("expected cache miss when cache is disabled")
	}
}

func TestTransformCacheBelowThreshold(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  10000, // High threshold
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	// Small data that's below threshold
	testData := make(nat, 10) // 640 bits on 64-bit system
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}},
	}

	// Put should be a no-op for data below threshold
	cache.Put(testData, mockValues)

	// Get should return false for data below threshold
	_, found := cache.Get(testData, 4, 100)
	if found {
		t.Error("expected cache miss for data below MinBitLen threshold")
	}
}

func TestTransformCacheStats(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}},
	}

	// Initial stats
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("expected zero hits and misses initially")
	}

	// Miss
	cache.Get(testData, 4, 100)
	stats = cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}

	// Put and hit
	cache.Put(testData, mockValues)
	cache.Get(testData, 4, 100)
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
}

func TestTransformCacheClear(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 10,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: []fermat{{1, 2, 3}},
	}

	cache.Put(testData, mockValues)
	cache.Get(testData, 4, 100)

	// Clear the cache
	cache.Clear()

	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("expected size=0 after Clear, got %d", stats.Size)
	}
	if stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
		t.Error("expected all stats to be zero after Clear")
	}

	// Verify entry is gone
	_, found := cache.Get(testData, 4, 100)
	if found {
		t.Error("expected cache miss after Clear")
	}
}

func TestTransformCacheConcurrency(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 100,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numOperations; i++ {
				testData := make(nat, 10)
				testData[0] = big.Word(goroutineID)
				testData[1] = big.Word(i % 10) // Some overlap

				mockValues := PolValues{
					K:      4,
					N:      100,
					Values: []fermat{{big.Word(goroutineID), big.Word(i)}},
				}

				// Mix of puts and gets
				if i%2 == 0 {
					cache.Put(testData, mockValues)
				} else {
					cache.Get(testData, 4, 100)
				}
			}
		}(g)
	}

	wg.Wait()

	// Just verify no panics occurred and stats are accessible
	stats := cache.Stats()
	t.Logf("Concurrency test stats: hits=%d, misses=%d, size=%d",
		stats.Hits, stats.Misses, stats.Size)
}

func TestComputeKeyConsistency(t *testing.T) {
	t.Parallel()
	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	key1 := computeKey(testData, 4, 100)
	key2 := computeKey(testData, 4, 100)

	if key1 != key2 {
		t.Error("computeKey should return consistent results for same input")
	}
}

func TestComputeKeyDifferentParams(t *testing.T) {
	t.Parallel()
	testData := make(nat, 10)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	key1 := computeKey(testData, 4, 100)
	key2 := computeKey(testData, 5, 100) // Different k
	key3 := computeKey(testData, 4, 200) // Different n

	if key1 == key2 {
		t.Error("different k values should produce different keys")
	}
	if key1 == key3 {
		t.Error("different n values should produce different keys")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Integration Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestCachedTransformIntegration(t *testing.T) {
	// Verify that cached transforms produce correct results
	x := new(big.Int).SetInt64(12345678901234567)
	y := new(big.Int).SetInt64(98765432109876543)

	// First multiplication (populates cache if applicable)
	result1, err := Mul(x, y)
	if err != nil {
		t.Fatalf("first Mul failed: %v", err)
	}

	// Second multiplication (may use cache)
	result2, err := Mul(x, y)
	if err != nil {
		t.Fatalf("second Mul failed: %v", err)
	}

	if result1.Cmp(result2) != 0 {
		t.Error("cached and non-cached results should be equal")
	}

	// Verify against big.Int.Mul
	expected := new(big.Int).Mul(x, y)
	if result1.Cmp(expected) != 0 {
		t.Errorf("result mismatch: got %s, expected %s", result1, expected)
	}
}

func TestCachedSquareIntegration(t *testing.T) {
	x := new(big.Int).SetInt64(12345678901234567)

	// First squaring
	result1, err := Sqr(x)
	if err != nil {
		t.Fatalf("first Sqr failed: %v", err)
	}

	// Second squaring (may use cache)
	result2, err := Sqr(x)
	if err != nil {
		t.Fatalf("second Sqr failed: %v", err)
	}

	if result1.Cmp(result2) != 0 {
		t.Error("cached and non-cached results should be equal")
	}

	// Verify against big.Int.Mul
	expected := new(big.Int).Mul(x, x)
	if result1.Cmp(expected) != 0 {
		t.Errorf("result mismatch: got %s, expected %s", result1, expected)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkCacheHit(b *testing.B) {
	b.ReportAllocs()
	config := TransformCacheConfig{
		MaxEntries: 100,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	testData := make(nat, 100)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: make([]fermat, 16),
	}
	for i := range mockValues.Values {
		mockValues.Values[i] = make(fermat, 101)
	}

	cache.Put(testData, mockValues)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(testData, 4, 100)
	}
}

func BenchmarkCacheMiss(b *testing.B) {
	b.ReportAllocs()
	config := TransformCacheConfig{
		MaxEntries: 100,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	testData := make(nat, 100)
	for i := range testData {
		testData[i] = big.Word(i + 1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(testData, 4, 100)
	}
}

func BenchmarkCachePut(b *testing.B) {
	b.ReportAllocs()
	config := TransformCacheConfig{
		MaxEntries: 1000,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	mockValues := PolValues{
		K:      4,
		N:      100,
		Values: make([]fermat, 16),
	}
	for i := range mockValues.Values {
		mockValues.Values[i] = make(fermat, 101)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testData := make(nat, 100)
		testData[0] = big.Word(i)
		cache.Put(testData, mockValues)
	}
}
