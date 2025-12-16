package bigfft

import (
	"math/big"
	"sync"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Unit Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestTransformCache_BasicOperations(t *testing.T) {
	cache := NewTransformCache(10)

	// Create a simple nat for testing
	x := nat{1, 2, 3, 4, 5}
	k := uint(4)
	n := 10

	// Create a mock polValues
	pv := &polValues{
		k: k,
		n: n,
		values: []fermat{
			{1, 2, 3},
			{4, 5, 6},
		},
	}

	// Test Put and Get
	cache.Put(x, k, n, pv)

	got := cache.Get(x, k, n)
	if got == nil {
		t.Fatal("expected cached value, got nil")
	}
	if got.k != k || got.n != n {
		t.Errorf("got k=%d n=%d, want k=%d n=%d", got.k, got.n, k, n)
	}
	if len(got.values) != len(pv.values) {
		t.Errorf("got %d values, want %d", len(got.values), len(pv.values))
	}
}

func TestTransformCache_Miss(t *testing.T) {
	cache := NewTransformCache(10)

	x := nat{1, 2, 3}
	got := cache.Get(x, 4, 10)
	if got != nil {
		t.Errorf("expected nil for cache miss, got %v", got)
	}
}

func TestTransformCache_LRUEviction(t *testing.T) {
	cache := NewTransformCache(2)

	x1 := nat{1}
	x2 := nat{2}
	x3 := nat{3}

	pv := &polValues{k: 4, n: 10, values: []fermat{{1}}}

	cache.Put(x1, 4, 10, pv)
	cache.Put(x2, 4, 10, pv)

	// Access x1 to make it recently used
	cache.Get(x1, 4, 10)

	// Add x3, should evict x2 (least recently used)
	cache.Put(x3, 4, 10, pv)

	if cache.Get(x2, 4, 10) != nil {
		t.Error("expected x2 to be evicted")
	}
	if cache.Get(x1, 4, 10) == nil {
		t.Error("expected x1 to still be present")
	}
	if cache.Get(x3, 4, 10) == nil {
		t.Error("expected x3 to still be present")
	}
}

func TestTransformCache_Stats(t *testing.T) {
	cache := NewTransformCache(10)

	x := nat{1, 2, 3}
	pv := &polValues{k: 4, n: 10, values: []fermat{{1}}}
	cache.Put(x, 4, 10, pv)

	cache.Get(x, 4, 10)      // hit
	cache.Get(x, 4, 10)      // hit
	cache.Get(nat{9}, 4, 10) // miss

	hits, misses, size := cache.Stats()
	if hits != 2 {
		t.Errorf("hits = %d, want 2", hits)
	}
	if misses != 1 {
		t.Errorf("misses = %d, want 1", misses)
	}
	if size != 1 {
		t.Errorf("size = %d, want 1", size)
	}
}

func TestTransformCache_ValueIsolation(t *testing.T) {
	cache := NewTransformCache(10)

	x := nat{1, 2, 3}
	original := &polValues{
		k: 4,
		n: 10,
		values: []fermat{
			{100, 200, 300},
		},
	}
	cache.Put(x, 4, 10, original)

	// Modify original - should not affect cached value
	original.values[0][0] = 999

	got := cache.Get(x, 4, 10)
	if got.values[0][0] != 100 {
		t.Error("cache value was mutated by external change")
	}

	// Modify retrieved value - should not affect cache
	got.values[0][0] = 888
	got2 := cache.Get(x, 4, 10)
	if got2.values[0][0] != 100 {
		t.Error("cache value was mutated by retrieved value change")
	}
}

func TestTransformCache_EnableDisable(t *testing.T) {
	cache := NewTransformCache(10)

	x := nat{1, 2, 3}
	pv := &polValues{k: 4, n: 10, values: []fermat{{1}}}

	// Disable cache
	cache.SetEnabled(false)
	cache.Put(x, 4, 10, pv)

	if cache.Get(x, 4, 10) != nil {
		t.Error("expected nil when cache is disabled")
	}

	// Re-enable
	cache.SetEnabled(true)
	cache.Put(x, 4, 10, pv)

	if cache.Get(x, 4, 10) == nil {
		t.Error("expected value when cache is re-enabled")
	}
}

func TestHashNat(t *testing.T) {
	// Same content should produce same hash
	x1 := nat{1, 2, 3, 4, 5}
	x2 := nat{1, 2, 3, 4, 5}
	if hashNat(x1) != hashNat(x2) {
		t.Error("same content should produce same hash")
	}

	// Different content should (usually) produce different hash
	x3 := nat{5, 4, 3, 2, 1}
	if hashNat(x1) == hashNat(x3) {
		t.Error("different content produced same hash (unlikely)")
	}

	// Empty nat
	empty := nat{}
	if hashNat(empty) != 0 {
		t.Error("empty nat should hash to 0")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Concurrency Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestTransformCache_Concurrency(t *testing.T) {
	cache := NewTransformCache(100)
	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				x := nat{big.Word(id*numOperations + j)}
				pv := &polValues{k: 4, n: 10, values: []fermat{{big.Word(j)}}}
				cache.Put(x, 4, 10, pv)
				cache.Get(x, 4, 10)
			}
		}(i)
	}
	wg.Wait()

	// Verify no panics occurred and cache is consistent
	if cache.lru.Len() > 100 {
		t.Errorf("cache exceeded capacity: %d > 100", cache.lru.Len())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkTransformCache_Get(b *testing.B) {
	cache := NewTransformCache(1000)
	x := nat{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	pv := &polValues{k: 4, n: 10, values: []fermat{{1, 2, 3}}}
	cache.Put(x, 4, 10, pv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(x, 4, 10)
	}
}

func BenchmarkTransformCache_Put(b *testing.B) {
	cache := NewTransformCache(1000)
	pv := &polValues{k: 4, n: 10, values: []fermat{{1, 2, 3}}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := nat{big.Word(i % 1000)}
		cache.Put(x, 4, 10, pv)
	}
}

func BenchmarkHashNat(b *testing.B) {
	x := make(nat, 1000)
	for i := range x {
		x[i] = big.Word(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hashNat(x)
	}
}
