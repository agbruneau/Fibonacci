package fibonacci

import (
	"math/big"
	"sync"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Unit Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestFibonacciCache_BasicOperations(t *testing.T) {
	cache := NewFibonacciCache(10)

	// Test Put and Get
	value := big.NewInt(55)
	cache.Put(10, value)

	got := cache.Get(10)
	if got == nil {
		t.Fatal("expected value, got nil")
	}
	if got.Cmp(value) != 0 {
		t.Errorf("got %v, want %v", got, value)
	}

	// Test cache miss
	miss := cache.Get(999)
	if miss != nil {
		t.Errorf("expected nil for cache miss, got %v", miss)
	}
}

func TestFibonacciCache_LRUEviction(t *testing.T) {
	cache := NewFibonacciCache(3)

	// Fill cache
	cache.Put(1, big.NewInt(1))
	cache.Put(2, big.NewInt(1))
	cache.Put(3, big.NewInt(2))

	// Access 1 to make it recently used
	cache.Get(1)

	// Add 4th element, should evict 2 (least recently used)
	cache.Put(4, big.NewInt(3))

	if cache.Get(2) != nil {
		t.Error("expected entry 2 to be evicted")
	}
	if cache.Get(1) == nil {
		t.Error("expected entry 1 to still be present")
	}
	if cache.Get(3) == nil {
		t.Error("expected entry 3 to still be present")
	}
	if cache.Get(4) == nil {
		t.Error("expected entry 4 to still be present")
	}
}

func TestFibonacciCache_Stats(t *testing.T) {
	cache := NewFibonacciCache(10)
	cache.Put(1, big.NewInt(1))

	cache.Get(1) // hit
	cache.Get(1) // hit
	cache.Get(2) // miss

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

func TestFibonacciCache_HitRate(t *testing.T) {
	cache := NewFibonacciCache(10)

	// Empty cache should have 0 hit rate
	if rate := cache.HitRate(); rate != 0.0 {
		t.Errorf("empty cache hit rate = %f, want 0.0", rate)
	}

	cache.Put(1, big.NewInt(1))
	cache.Get(1) // hit
	cache.Get(2) // miss

	rate := cache.HitRate()
	if rate != 0.5 {
		t.Errorf("hit rate = %f, want 0.5", rate)
	}
}

func TestFibonacciCache_Clear(t *testing.T) {
	cache := NewFibonacciCache(10)
	cache.Put(1, big.NewInt(1))
	cache.Put(2, big.NewInt(1))
	cache.Get(1) // hit

	cache.Clear()

	if cache.Size() != 0 {
		t.Error("cache should be empty after Clear")
	}
	hits, misses, _ := cache.Stats()
	if hits != 0 || misses != 0 {
		t.Error("stats should be reset after Clear")
	}
}

func TestFibonacciCache_ValueIsolation(t *testing.T) {
	cache := NewFibonacciCache(10)
	original := big.NewInt(55)
	cache.Put(10, original)

	// Modify original - should not affect cached value
	original.SetInt64(999)

	got := cache.Get(10)
	if got.Cmp(big.NewInt(55)) != 0 {
		t.Error("cache value was mutated by external change to original")
	}

	// Modify retrieved value - should not affect cache
	got.SetInt64(888)
	got2 := cache.Get(10)
	if got2.Cmp(big.NewInt(55)) != 0 {
		t.Error("cache value was mutated by external change to retrieved value")
	}
}

func TestFibonacciCache_Update(t *testing.T) {
	cache := NewFibonacciCache(10)
	cache.Put(1, big.NewInt(100))
	cache.Put(1, big.NewInt(200))

	got := cache.Get(1)
	if got.Cmp(big.NewInt(200)) != 0 {
		t.Errorf("got %v, want 200", got)
	}

	if cache.Size() != 1 {
		t.Errorf("size = %d, want 1 (update should not add new entry)", cache.Size())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Concurrency Tests
// ─────────────────────────────────────────────────────────────────────────────

func TestFibonacciCache_Concurrency(t *testing.T) {
	cache := NewFibonacciCache(100)
	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 1000

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				n := uint64((id*numOperations + j) % 200)
				cache.Put(n, big.NewInt(int64(n)))
				cache.Get(n)
			}
		}(i)
	}
	wg.Wait()

	// Just verify no data races occurred
	if cache.Size() > 100 {
		t.Errorf("cache exceeded capacity: %d > 100", cache.Size())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

func BenchmarkFibonacciCache_Get(b *testing.B) {
	cache := NewFibonacciCache(1000)
	for i := uint64(0); i < 1000; i++ {
		cache.Put(i, big.NewInt(int64(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(uint64(i % 1000))
	}
}

func BenchmarkFibonacciCache_Put(b *testing.B) {
	cache := NewFibonacciCache(1000)
	value := big.NewInt(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Put(uint64(i%1000), value)
	}
}

func BenchmarkFibonacciCache_ConcurrentAccess(b *testing.B) {
	cache := NewFibonacciCache(1000)
	for i := uint64(0); i < 1000; i++ {
		cache.Put(i, big.NewInt(int64(i)))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := uint64(0)
		for pb.Next() {
			cache.Get(i % 1000)
			i++
		}
	})
}
