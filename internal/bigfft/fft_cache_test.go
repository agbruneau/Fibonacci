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
	if cache.config.MaxEntries != 256 {
		t.Errorf("expected MaxEntries=256, got %d", cache.config.MaxEntries)
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
		// Note: MinBitLen must be lower than testData size for Put to accept the entry
		// testData has 2000 words = 2000 * 64 bits = 128000 bits on 64-bit systems
		config1 := TransformCacheConfig{
			MaxEntries: 64,
			MinBitLen:  1000, // Low threshold to ensure our test data is accepted
			Enabled:    true,
		}
		SetTransformCacheConfig(config1)

		cache := GetTransformCache()
		// Add a real entry using Put
		// Need enough words to exceed MinBitLen (1000 bits = ~16 words on 64-bit)
		testData := make(nat, 2000) // 2000 words = 128000 bits on 64-bit
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

// TestTransformCacheEvictionRecyclesBacking historically verified the
// [R1.5] backing-recycle optimisation. Per Audit-PRD E1-R4, the recycle
// was removed because it opened a use-after-free window for callers still
// holding a PolValues returned by a previous Get(). The test is retained
// as a *steady-state allocation upper bound*: at cache capacity the put
// path should still keep allocations bounded, just not recycle.
//
// Not marked t.Parallel(): testing.AllocsPerRun panics in parallel tests.
func TestTransformCacheEvictionRecyclesBacking(t *testing.T) {
	const maxEntries = 4
	config := TransformCacheConfig{
		MaxEntries: maxEntries,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	// All entries share the same shape (K=8 coefficients of length n+1=33).
	// The backing buffer is K*(n+1) = 264 big.Words = 2112 bytes on 64-bit.
	const K, N = 8, 32

	makeValues := func(seed int) PolValues {
		vals := make([]fermat, K)
		for i := range vals {
			vals[i] = make(fermat, N+1)
			vals[i][0] = big.Word(seed*K + i)
		}
		return PolValues{K: 4, N: N, Values: vals}
	}

	// Pre-fill the cache to exactly MaxEntries with distinct keys.
	for i := 0; i < maxEntries; i++ {
		data := make(nat, 10)
		data[0] = big.Word(0xA000 + i)
		cache.Put(data, makeValues(i))
	}
	if got := cache.Stats().Size; got != maxEntries {
		t.Fatalf("pre-fill: expected size=%d, got %d", maxEntries, got)
	}

	// Counter so each AllocsPerRun iteration uses a fresh key (otherwise
	// putByKey short-circuits on the duplicate-key check and skips the
	// allocation path under test).
	counter := 0
	avg := testing.AllocsPerRun(50, func() {
		counter++
		data := make(nat, 10)
		data[0] = big.Word(0xB000 + counter)
		cache.Put(data, makeValues(counter))
	})

	// Allocations attributable to Put at steady state (cache full, all inserts
	// trigger eviction):
	//   1) the data slice we build outside cache.Put (counted by AllocsPerRun)
	//   2) PolValues.Values slice + each fermat backing (counted; built outside
	//      cache.Put as test fixture)
	//   3) cache-internal: cacheEntry struct, valuesCopy []fermat header
	//
	// Crucially, the K*(n+1) big.Word backing buffer must NOT be reallocated
	// because the recycled evicted buffer fits. Without the [R1.5] fix, a
	// fresh `make([]big.Word, 264)` would add ~1 alloc per call. With it,
	// allocations stay below an empirical bound.
	//
	// Test fixture allocs (data + PolValues.Values + K fermats) = 1 + 1 + K = 10.
	// Cache-internal allocs (cacheEntry + valuesCopy header + list element) ~= 3.
	// Conservative ceiling: 16 allocs. Without recycling, we'd see ~14+1=15
	// PLUS the salient 264-word backing alloc, but the count alone wouldn't
	// catch it; we therefore also probe the cap of evicted backings via a
	// direct check below.
	const allocCeiling = 16.0
	if avg > allocCeiling {
		t.Errorf("Put at steady state: avg allocs/op = %.1f, want <= %.1f "+
			"(R1.5 backing recycling regressed?)", avg, allocCeiling)
	}

	// Direct invariant check: after several evictions, every live entry's
	// backing buffer must have capacity exactly K*(N+1) and must be one of
	// the originals. We probe by inserting one more entry and confirming the
	// new entry's backing capacity equals the recycled buffer's capacity.
	cache.mu.Lock()
	var seenCap int
	for _, elem := range cache.entries {
		e := elem.Value.(*cacheEntry)
		if seenCap == 0 {
			seenCap = cap(e.backing)
		} else if cap(e.backing) != seenCap {
			cache.mu.Unlock()
			t.Fatalf("entries have heterogeneous backing capacities: %d vs %d",
				seenCap, cap(e.backing))
		}
	}
	cache.mu.Unlock()

	if seenCap != K*(N+1) {
		t.Errorf("expected backing cap = %d, got %d", K*(N+1), seenCap)
	}

	// Sanity: evictions actually occurred.
	if ev := cache.Stats().Evictions; ev == 0 {
		t.Error("expected evictions during steady-state inserts")
	}
}

// TestTransformCacheEvictionAllocsFreshAllocWhenTooSmall verifies the fallback
// path: if the evicted backing is too small for the new entry, putByKey must
// allocate fresh storage (and the new entry's cap must reflect the new size).
func TestTransformCacheEvictionAllocsFreshAllocWhenTooSmall(t *testing.T) {
	t.Parallel()
	config := TransformCacheConfig{
		MaxEntries: 2,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	// Fill with small entries.
	for i := 0; i < 2; i++ {
		data := make(nat, 10)
		data[0] = big.Word(i)
		small := PolValues{
			K: 4, N: 4,
			Values: []fermat{make(fermat, 5), make(fermat, 5)},
		}
		cache.Put(data, small)
	}

	// Insert a much larger entry — cannot reuse the small backings.
	bigData := make(nat, 10)
	bigData[0] = big.Word(0xDEAD)
	const bigK, bigN = 16, 64
	bigVals := make([]fermat, bigK)
	for i := range bigVals {
		bigVals[i] = make(fermat, bigN+1)
	}
	cache.Put(bigData, PolValues{K: 8, N: bigN, Values: bigVals})

	cache.mu.Lock()
	defer cache.mu.Unlock()
	for _, elem := range cache.entries {
		e := elem.Value.(*cacheEntry)
		if e.n == bigN {
			if cap(e.backing) < bigK*(bigN+1) {
				t.Errorf("large entry backing cap %d < required %d",
					cap(e.backing), bigK*(bigN+1))
			}
			return
		}
	}
	t.Fatal("large entry not found in cache")
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

// BenchmarkCacheHitParallel measures the contention of cache hits under
// concurrent load. This is the hot-path scenario targeted by R3.9: with a
// single hot key, every goroutine races on the same elem and would, under
// the old implementation, contend on the exclusive Lock used for
// MoveToFront. The R3.9 fast path detects "already at front" under RLock
// and skips the exclusive Lock entirely, dramatically reducing contention.
func BenchmarkCacheHitParallel(b *testing.B) {
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
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get(testData, 4, 100)
		}
	})
}

// BenchmarkTransformCache exposes the suite of micro-benchmarks for the
// transform cache hot path under a single canonical name (used by R3.9
// validation). It runs the parallel hit benchmark — the scenario where
// the optimization matters most.
func BenchmarkTransformCache(b *testing.B) {
	b.Run("HitParallel", BenchmarkCacheHitParallel)
	b.Run("Hit", BenchmarkCacheHit)
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

// ─────────────────────────────────────────────────────────────────────────────
// Hash invariance regression test (R2.10)
//
// computeCacheKey and computePolyKey were refactored to share a single
// FNV-1a implementation via cacheKeyBuilder. Since FFT cache lookups depend
// on the exact bit-pattern of the produced keys, any change to the hash
// algorithm would silently invalidate previously hot cache entries between
// rebuilds — and, worse, between callers within the same process if some
// path were missed in the refactor. This test pins the algorithm to the
// pre-refactor reference (an inline FNV-1a) and verifies bit-equality.
// ─────────────────────────────────────────────────────────────────────────────

// referenceFnvWriteUint64 is a verbatim copy of the pre-R2.10 fnvWriteUint64
// helper. It serves as the oracle for the consolidated implementation.
func referenceFnvWriteUint64(h uint64, x uint64) uint64 {
	h ^= x & 0xff
	h *= prime64
	h ^= (x >> 8) & 0xff
	h *= prime64
	h ^= (x >> 16) & 0xff
	h *= prime64
	h ^= (x >> 24) & 0xff
	h *= prime64
	h ^= (x >> 32) & 0xff
	h *= prime64
	h ^= (x >> 40) & 0xff
	h *= prime64
	h ^= (x >> 48) & 0xff
	h *= prime64
	h ^= (x >> 56) & 0xff
	h *= prime64
	return h
}

// referenceComputeCacheKey is a verbatim copy of the pre-R2.10 algorithm.
func referenceComputeCacheKey(data nat, k uint, n int) uint64 {
	h := uint64(offset64)
	h = referenceFnvWriteUint64(h, uint64(k))
	h = referenceFnvWriteUint64(h, uint64(n))
	for _, word := range data {
		h = referenceFnvWriteUint64(h, uint64(word))
	}
	return h
}

// referenceComputePolyKey is a verbatim copy of the pre-R2.10 algorithm.
func referenceComputePolyKey(p *Poly, k uint, n int) uint64 {
	h := uint64(offset64)
	h = referenceFnvWriteUint64(h, uint64(k))
	h = referenceFnvWriteUint64(h, uint64(n))
	for _, a := range p.A {
		for _, word := range a {
			h = referenceFnvWriteUint64(h, uint64(word))
		}
	}
	return h
}

func TestComputeCacheKeyInvariantR2_10(t *testing.T) {
	t.Parallel()

	// Cover edge cases: empty, single word, multiple words, large words.
	cases := []struct {
		name string
		data nat
		k    uint
		n    int
	}{
		{"empty", nat{}, 0, 0},
		{"single_zero", nat{0}, 1, 1},
		{"single_max", nat{^big.Word(0)}, 7, 13},
		{"sequential", func() nat {
			d := make(nat, 16)
			for i := range d {
				d[i] = big.Word(i + 1)
			}
			return d
		}(), 4, 100},
		{"large_k_n", nat{0xdeadbeef, 0xcafebabe, 0x12345678}, 1 << 20, 1 << 30},
	}

	for _, tc := range cases {
		got := computeCacheKey(tc.data, tc.k, tc.n)
		want := referenceComputeCacheKey(tc.data, tc.k, tc.n)
		if got != want {
			t.Errorf("%s: computeCacheKey hash drift: got=%#x want=%#x", tc.name, got, want)
		}
	}
}

func TestComputePolyKeyInvariantR2_10(t *testing.T) {
	t.Parallel()

	makePoly := func(coeffs [][]big.Word) *Poly {
		p := &Poly{A: make([]nat, len(coeffs))}
		for i, c := range coeffs {
			p.A[i] = nat(c)
		}
		return p
	}

	cases := []struct {
		name string
		poly *Poly
		k    uint
		n    int
	}{
		{"empty_poly", makePoly(nil), 0, 0},
		{"single_empty_coeff", makePoly([][]big.Word{{}}), 1, 1},
		{"uniform_small", makePoly([][]big.Word{
			{1, 2, 3},
			{4, 5, 6},
			{7, 8, 9},
		}), 3, 27},
		{"jagged", makePoly([][]big.Word{
			{0xdeadbeef},
			{},
			{1, 2, 3, 4, 5, 6, 7, 8},
			{^big.Word(0)},
		}), 5, 200},
	}

	for _, tc := range cases {
		got := computePolyKey(tc.poly, tc.k, tc.n)
		want := referenceComputePolyKey(tc.poly, tc.k, tc.n)
		if got != want {
			t.Errorf("%s: computePolyKey hash drift: got=%#x want=%#x", tc.name, got, want)
		}
	}
}
