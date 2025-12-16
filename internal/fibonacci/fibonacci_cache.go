// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file implements a thread-safe LRU cache for Fibonacci results.
package fibonacci

import (
	"container/list"
	"math/big"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Cache Configuration
// ─────────────────────────────────────────────────────────────────────────────

const (
	// DefaultCacheCapacity is the default maximum number of entries in the cache.
	// Chosen to balance memory usage with cache hit probability for typical
	// Fibonacci list calculations.
	DefaultCacheCapacity = 1000

	// DefaultCacheEnabled controls whether caching is enabled by default.
	DefaultCacheEnabled = true
)

// ─────────────────────────────────────────────────────────────────────────────
// Cache Implementation
// ─────────────────────────────────────────────────────────────────────────────

// cacheEntry represents a single cached Fibonacci value.
type cacheEntry struct {
	n     uint64
	value *big.Int
}

// FibonacciCache provides a thread-safe LRU cache for Fibonacci results.
// It significantly improves performance when calculating lists of Fibonacci
// numbers or when the same values are requested multiple times.
//
// The cache uses a doubly-linked list for O(1) LRU eviction and a map for
// O(1) lookups, providing efficient access patterns.
type FibonacciCache struct {
	mu       sync.RWMutex
	capacity int
	cache    map[uint64]*list.Element
	lru      *list.List
	hits     uint64
	misses   uint64
}

// NewFibonacciCache creates a new cache with the specified capacity.
// If capacity <= 0, DefaultCacheCapacity is used.
//
// Parameters:
//   - capacity: Maximum number of entries to store.
//
// Returns:
//   - *FibonacciCache: A new, empty cache.
func NewFibonacciCache(capacity int) *FibonacciCache {
	if capacity <= 0 {
		capacity = DefaultCacheCapacity
	}
	return &FibonacciCache{
		capacity: capacity,
		cache:    make(map[uint64]*list.Element, capacity),
		lru:      list.New(),
	}
}

// Get retrieves a cached Fibonacci value for the given index.
// Returns nil if the value is not in the cache.
//
// This operation is thread-safe and promotes the accessed entry to the
// front of the LRU list.
//
// Parameters:
//   - n: The Fibonacci index to look up.
//
// Returns:
//   - *big.Int: The cached value, or nil if not found.
func (c *FibonacciCache) Get(n uint64) *big.Int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[n]; ok {
		c.hits++
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		// Return a copy to prevent external mutation
		return new(big.Int).Set(entry.value)
	}
	c.misses++
	return nil
}

// Put stores a Fibonacci value in the cache.
// If the cache is at capacity, the least recently used entry is evicted.
//
// This operation is thread-safe.
//
// Parameters:
//   - n: The Fibonacci index.
//   - value: The Fibonacci value to cache.
func (c *FibonacciCache) Put(n uint64, value *big.Int) {
	if value == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If already present, update and move to front
	if elem, ok := c.cache[n]; ok {
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.value = new(big.Int).Set(value)
		return
	}

	// Evict LRU if at capacity
	if c.lru.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new entry
	entry := &cacheEntry{
		n:     n,
		value: new(big.Int).Set(value),
	}
	elem := c.lru.PushFront(entry)
	c.cache[n] = elem
}

// evictOldest removes the least recently used entry from the cache.
// Must be called with the mutex held.
func (c *FibonacciCache) evictOldest() {
	if oldest := c.lru.Back(); oldest != nil {
		entry := oldest.Value.(*cacheEntry)
		delete(c.cache, entry.n)
		c.lru.Remove(oldest)
	}
}

// Stats returns cache hit/miss statistics.
//
// Returns:
//   - hits: Number of successful cache lookups.
//   - misses: Number of failed cache lookups.
//   - size: Current number of entries in cache.
func (c *FibonacciCache) Stats() (hits, misses uint64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.lru.Len()
}

// HitRate returns the cache hit rate as a value between 0.0 and 1.0.
// Returns 0.0 if no lookups have been performed.
func (c *FibonacciCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}
	return float64(c.hits) / float64(total)
}

// Clear removes all entries from the cache and resets statistics.
func (c *FibonacciCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[uint64]*list.Element, c.capacity)
	c.lru.Init()
	c.hits = 0
	c.misses = 0
}

// Size returns the current number of entries in the cache.
func (c *FibonacciCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// ─────────────────────────────────────────────────────────────────────────────
// Global Cache Instance
// ─────────────────────────────────────────────────────────────────────────────

// globalCache is the default shared cache instance.
var globalCache *FibonacciCache
var globalCacheOnce sync.Once

// GetGlobalCache returns the singleton global cache instance.
// The cache is lazily initialized on first access.
func GetGlobalCache() *FibonacciCache {
	globalCacheOnce.Do(func() {
		globalCache = NewFibonacciCache(DefaultCacheCapacity)
	})
	return globalCache
}

// ResetGlobalCache clears and reinitializes the global cache.
// This is primarily useful for testing.
func ResetGlobalCache() {
	globalCacheOnce.Do(func() {
		globalCache = NewFibonacciCache(DefaultCacheCapacity)
	})
	globalCache.Clear()
}
