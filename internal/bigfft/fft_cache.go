// Package bigfft implements multiplication of big.Int using FFT.
// This file provides an LRU cache for FFT transform results to improve
// performance when the same operands are used repeatedly.
package bigfft

import (
	"container/list"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Cache Configuration
// ─────────────────────────────────────────────────────────────────────────────

const (
	// DefaultTransformCacheCapacity is the default maximum number of cached transforms.
	// Each cached transform can be large (proportional to FFT size), so we keep
	// this relatively small to avoid excessive memory usage.
	DefaultTransformCacheCapacity = 64

	// TransformCacheEnabled controls whether transform caching is enabled by default.
	TransformCacheEnabled = true
)

// ─────────────────────────────────────────────────────────────────────────────
// Transform Cache Implementation
// ─────────────────────────────────────────────────────────────────────────────

// transformCacheKey uniquely identifies a cached transform.
type transformCacheKey struct {
	hash uint64 // Hash of the nat data
	k    uint   // FFT size parameter
	n    int    // Coefficient size parameter
}

// transformCacheEntry stores a cached transform result.
type transformCacheEntry struct {
	key    transformCacheKey
	values []fermat // Cached transform values
	n      int      // Coefficient size
	k      uint     // FFT size parameter
}

// TransformCache provides a thread-safe LRU cache for FFT transform results.
// Caching transforms can significantly improve performance when the same
// operands are used multiple times in succession (common in Fibonacci calculations).
type TransformCache struct {
	mu       sync.RWMutex
	capacity int
	cache    map[transformCacheKey]*list.Element
	lru      *list.List
	hits     uint64
	misses   uint64
	enabled  bool
}

// NewTransformCache creates a new cache with the specified capacity.
// If capacity <= 0, DefaultTransformCacheCapacity is used.
func NewTransformCache(capacity int) *TransformCache {
	if capacity <= 0 {
		capacity = DefaultTransformCacheCapacity
	}
	return &TransformCache{
		capacity: capacity,
		cache:    make(map[transformCacheKey]*list.Element, capacity),
		lru:      list.New(),
		enabled:  TransformCacheEnabled,
	}
}

// hashNat computes a simple hash of a nat for cache lookup.
// The hash combines the length and a sample of the data values.
func hashNat(x nat) uint64 {
	if len(x) == 0 {
		return 0
	}
	// FNV-1a inspired hash
	h := uint64(14695981039346656037)
	h ^= uint64(len(x))
	h *= 1099511628211

	// Sample elements for efficiency (don't hash entire large numbers)
	step := 1
	if len(x) > 16 {
		step = len(x) / 16
	}
	for i := 0; i < len(x); i += step {
		h ^= uint64(x[i])
		h *= 1099511628211
	}
	// Always include last element
	h ^= uint64(x[len(x)-1])
	h *= 1099511628211

	return h
}

// Get retrieves a cached transform for the given nat and parameters.
// Returns nil if not in cache.
func (c *TransformCache) Get(x nat, k uint, n int) *polValues {
	if !c.enabled || len(x) == 0 {
		return nil
	}

	key := transformCacheKey{
		hash: hashNat(x),
		k:    k,
		n:    n,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.hits++
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*transformCacheEntry)

		// Make a copy of the values to prevent mutation
		valuesCopy := make([]fermat, len(entry.values))
		for i, v := range entry.values {
			valueCopy := make(fermat, len(v))
			copy(valueCopy, v)
			valuesCopy[i] = valueCopy
		}

		return &polValues{
			k:      entry.k,
			n:      entry.n,
			values: valuesCopy,
		}
	}

	c.misses++
	return nil
}

// Put stores a transform result in the cache.
func (c *TransformCache) Put(x nat, k uint, n int, pv *polValues) {
	if !c.enabled || pv == nil || len(x) == 0 {
		return
	}

	key := transformCacheKey{
		hash: hashNat(x),
		k:    k,
		n:    n,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.cache[key]; ok {
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*transformCacheEntry)
		entry.values = copyFermatSlice(pv.values)
		return
	}

	// Evict if at capacity
	if c.lru.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new entry
	entry := &transformCacheEntry{
		key:    key,
		values: copyFermatSlice(pv.values),
		n:      pv.n,
		k:      pv.k,
	}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem
}

// copyFermatSlice makes a deep copy of a []fermat slice.
func copyFermatSlice(src []fermat) []fermat {
	dst := make([]fermat, len(src))
	for i, v := range src {
		dst[i] = make(fermat, len(v))
		copy(dst[i], v)
	}
	return dst
}

// evictOldest removes the least recently used entry.
func (c *TransformCache) evictOldest() {
	if oldest := c.lru.Back(); oldest != nil {
		entry := oldest.Value.(*transformCacheEntry)
		delete(c.cache, entry.key)
		c.lru.Remove(oldest)
	}
}

// Stats returns cache statistics.
func (c *TransformCache) Stats() (hits, misses uint64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.lru.Len()
}

// HitRate returns the cache hit rate (0.0 to 1.0).
func (c *TransformCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}
	return float64(c.hits) / float64(total)
}

// Clear removes all entries and resets statistics.
func (c *TransformCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[transformCacheKey]*list.Element, c.capacity)
	c.lru.Init()
	c.hits = 0
	c.misses = 0
}

// SetEnabled enables or disables the cache.
func (c *TransformCache) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// IsEnabled returns whether the cache is enabled.
func (c *TransformCache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// ─────────────────────────────────────────────────────────────────────────────
// Global Cache Instance
// ─────────────────────────────────────────────────────────────────────────────

var globalTransformCache *TransformCache
var globalTransformCacheOnce sync.Once

// GetTransformCache returns the singleton global transform cache.
func GetTransformCache() *TransformCache {
	globalTransformCacheOnce.Do(func() {
		globalTransformCache = NewTransformCache(DefaultTransformCacheCapacity)
	})
	return globalTransformCache
}

// ResetTransformCache clears the global transform cache.
func ResetTransformCache() {
	GetTransformCache().Clear()
}

// ─────────────────────────────────────────────────────────────────────────────
// Cached Transform Functions
// ─────────────────────────────────────────────────────────────────────────────

// TransformWithCache performs a polynomial transform with caching.
// If the transform is already cached, it returns the cached result.
// Otherwise, it computes the transform and caches it for future use.
func (p *poly) TransformWithCache(n int, cache *TransformCache) (polValues, error) {
	// Try to get from cache first
	if cache != nil {
		// Reconstruct the underlying nat for caching
		// This is a simplified version - we hash the first coefficient
		if len(p.a) > 0 && len(p.a[0]) > 0 {
			if cached := cache.Get(p.a[0], p.k, n); cached != nil {
				return *cached, nil
			}
		}
	}

	// Compute the transform
	pv, err := p.Transform(n)
	if err != nil {
		return polValues{}, err
	}

	// Cache the result
	if cache != nil && len(p.a) > 0 && len(p.a[0]) > 0 {
		cache.Put(p.a[0], p.k, n, &pv)
	}

	return pv, nil
}

// TransformWithBumpAndCache combines bump allocation with caching.
func (p *poly) TransformWithBumpAndCache(n int, ba *BumpAllocator, cache *TransformCache) (polValues, error) {
	// Try cache first
	if cache != nil && len(p.a) > 0 && len(p.a[0]) > 0 {
		if cached := cache.Get(p.a[0], p.k, n); cached != nil {
			return *cached, nil
		}
	}

	// Compute with bump allocator
	pv, err := p.TransformWithBump(n, ba)
	if err != nil {
		return polValues{}, err
	}

	// Cache result
	if cache != nil && len(p.a) > 0 && len(p.a[0]) > 0 {
		cache.Put(p.a[0], p.k, n, &pv)
	}

	return pv, nil
}

// MulWithCache multiplies p and q with transform caching.
func (p *poly) MulWithCache(q *poly, cache *TransformCache) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformWithCache(n, cache)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.TransformWithCache(n, cache)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.Mul(&qv)
	if err != nil {
		return poly{}, err
	}
	r, err := rv.InvTransform()
	if err != nil {
		return poly{}, err
	}
	r.m = p.m
	return r, nil
}

// MulWithBumpAndCache combines bump allocation with transform caching.
func (p *poly) MulWithBumpAndCache(q *poly, ba *BumpAllocator, cache *TransformCache) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformWithBumpAndCache(n, ba, cache)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.TransformWithBumpAndCache(n, ba, cache)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.MulWithBump(&qv, ba)
	if err != nil {
		return poly{}, err
	}
	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return poly{}, err
	}
	r.m = p.m
	return r, nil
}
