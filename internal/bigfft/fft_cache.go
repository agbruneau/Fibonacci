// Package bigfft implements multiplication of big.Int using FFT.
// This file provides a thread-safe LRU cache for FFT transform results.
package bigfft

import (
	"container/list"
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// FFT Transform Cache
// ─────────────────────────────────────────────────────────────────────────────

// TransformCacheConfig holds configuration for the FFT transform cache.
type TransformCacheConfig struct {
	// MaxEntries is the maximum number of cached transforms.
	// Default: 128 entries
	MaxEntries int

	// MinBitLen is the minimum operand bit length to cache.
	// Smaller values don't benefit from caching.
	// Default: 100000 bits (~12KB)
	MinBitLen int

	// Enabled controls whether caching is active.
	// Default: true
	Enabled bool
}

// DefaultTransformCacheConfig returns the default cache configuration.
func DefaultTransformCacheConfig() TransformCacheConfig {
	return TransformCacheConfig{
		MaxEntries: 128,
		MinBitLen:  100000,
		Enabled:    true,
	}
}

// cacheEntry holds a cached FFT transform result.
type cacheEntry struct {
	key    [32]byte // SHA-256 hash of input
	values []fermat // cached polValues.values
	k      uint     // FFT size parameter
	n      int      // coefficient length
}

// TransformCache is a thread-safe LRU cache for FFT transforms.
// It caches the forward FFT transform results to avoid recomputation
// when the same values are multiplied repeatedly.
type TransformCache struct {
	mu        sync.RWMutex
	config    TransformCacheConfig
	entries   map[[32]byte]*list.Element
	lru       *list.List
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

// NewTransformCache creates a new FFT transform cache with the given config.
func NewTransformCache(config TransformCacheConfig) *TransformCache {
	return &TransformCache{
		config:  config,
		entries: make(map[[32]byte]*list.Element),
		lru:     list.New(),
	}
}

// globalTransformCache is the package-level transform cache.
var globalTransformCache *TransformCache
var transformCacheOnce sync.Once

// GetTransformCache returns the global FFT transform cache.
func GetTransformCache() *TransformCache {
	transformCacheOnce.Do(func() {
		globalTransformCache = NewTransformCache(DefaultTransformCacheConfig())
	})
	return globalTransformCache
}

// SetTransformCacheConfig updates the global cache configuration.
// This should be called before any FFT operations for consistent behavior.
func SetTransformCacheConfig(config TransformCacheConfig) {
	cache := GetTransformCache()
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.config = config

	// Optionally clear cache if disabled
	if !config.Enabled {
		cache.entries = make(map[[32]byte]*list.Element)
		cache.lru.Init()
	}
}

// computeKey generates a cache key from the input data.
// Uses SHA-256 for collision resistance with large numbers.
func computeKey(data nat, k uint, n int) [32]byte {
	h := sha256.New()

	// Include FFT parameters in the key
	var params [16]byte
	binary.LittleEndian.PutUint64(params[0:8], uint64(k))
	binary.LittleEndian.PutUint64(params[8:16], uint64(n))
	h.Write(params[:])

	// Write the number data
	buf := make([]byte, 8)
	for _, word := range data {
		binary.LittleEndian.PutUint64(buf, uint64(word))
		h.Write(buf)
	}

	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key
}

// Get retrieves a cached transform if available.
// Returns the polValues and true if found, zero values and false otherwise.
func (tc *TransformCache) Get(data nat, k uint, n int) (polValues, bool) {
	if !tc.config.Enabled || len(data)*_W < tc.config.MinBitLen {
		return polValues{}, false
	}

	key := computeKey(data, k, n)

	tc.mu.RLock()
	elem, found := tc.entries[key]
	tc.mu.RUnlock()

	if !found {
		tc.misses.Add(1)
		return polValues{}, false
	}

	tc.mu.Lock()
	tc.lru.MoveToFront(elem)
	tc.mu.Unlock()

	tc.hits.Add(1)

	entry := elem.Value.(*cacheEntry)

	// Return a copy of the cached values to avoid data races
	valuesCopy := make([]fermat, len(entry.values))
	for i, v := range entry.values {
		c := make(fermat, len(v))
		copy(c, v)
		valuesCopy[i] = c
	}

	return polValues{
		k:      entry.k,
		n:      entry.n,
		values: valuesCopy,
	}, true
}

// Put stores a transform result in the cache.
func (tc *TransformCache) Put(data nat, pv polValues) {
	if !tc.config.Enabled || len(data)*_W < tc.config.MinBitLen {
		return
	}

	key := computeKey(data, pv.k, pv.n)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Check if already cached
	if _, found := tc.entries[key]; found {
		return
	}

	// Evict oldest entries if at capacity
	for tc.lru.Len() >= tc.config.MaxEntries {
		oldest := tc.lru.Back()
		if oldest != nil {
			tc.lru.Remove(oldest)
			entry := oldest.Value.(*cacheEntry)
			delete(tc.entries, entry.key)
			tc.evictions.Add(1)
		}
	}

	// Create a deep copy of the values
	valuesCopy := make([]fermat, len(pv.values))
	for i, v := range pv.values {
		c := make(fermat, len(v))
		copy(c, v)
		valuesCopy[i] = c
	}

	entry := &cacheEntry{
		key:    key,
		values: valuesCopy,
		k:      pv.k,
		n:      pv.n,
	}

	elem := tc.lru.PushFront(entry)
	tc.entries[key] = elem
}

// Stats returns cache statistics.
type CacheStats struct {
	Hits      uint64
	Misses    uint64
	Evictions uint64
	Size      int
	HitRate   float64
}

// Stats returns current cache statistics.
func (tc *TransformCache) Stats() CacheStats {
	tc.mu.RLock()
	size := tc.lru.Len()
	tc.mu.RUnlock()

	hits := tc.hits.Load()
	misses := tc.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:      hits,
		Misses:    misses,
		Evictions: tc.evictions.Load(),
		Size:      size,
		HitRate:   hitRate,
	}
}

// Clear removes all entries from the cache.
func (tc *TransformCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.entries = make(map[[32]byte]*list.Element)
	tc.lru.Init()
	tc.hits.Store(0)
	tc.misses.Store(0)
	tc.evictions.Store(0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cached Transform Functions
// ─────────────────────────────────────────────────────────────────────────────

// TransformCached is like Transform but uses the global cache.
// If the transform result is cached, it returns the cached value.
// Otherwise, it computes the transform and caches the result.
func (p *poly) TransformCached(n int) (polValues, error) {
	cache := GetTransformCache()

	// Build a flat representation of p.a for key computation
	flatData := flattenPolyData(p)

	// Try cache lookup
	if cached, found := cache.Get(flatData, p.k, n); found {
		return cached, nil
	}

	// Compute transform
	pv, err := p.Transform(n)
	if err != nil {
		return polValues{}, err
	}

	// Cache the result
	cache.Put(flatData, pv)

	return pv, nil
}

// TransformCachedWithBump is like TransformWithBump but uses the global cache.
func (p *poly) TransformCachedWithBump(n int, ba *BumpAllocator) (polValues, error) {
	cache := GetTransformCache()

	// Build a flat representation of p.a for key computation
	flatData := flattenPolyData(p)

	// Try cache lookup
	if cached, found := cache.Get(flatData, p.k, n); found {
		return cached, nil
	}

	// Compute transform
	pv, err := p.TransformWithBump(n, ba)
	if err != nil {
		return polValues{}, err
	}

	// Cache the result
	cache.Put(flatData, pv)

	return pv, nil
}

// flattenPolyData creates a flat nat from polynomial coefficients for caching.
func flattenPolyData(p *poly) nat {
	totalLen := 0
	for _, a := range p.a {
		totalLen += len(a)
	}

	flat := make(nat, totalLen)
	offset := 0
	for _, a := range p.a {
		copy(flat[offset:], a)
		offset += len(a)
	}

	return flat
}

// MulCached multiplies p and q using cached transforms when beneficial.
func (p *poly) MulCached(q *poly) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformCached(n)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.TransformCached(n)
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

// MulCachedWithBump multiplies p and q using cached transforms and bump allocator.
func (p *poly) MulCachedWithBump(q *poly, ba *BumpAllocator) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		return poly{}, err
	}
	qv, err := q.TransformCachedWithBump(n, ba)
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

// SqrCached computes p*p using cached transform when beneficial.
func (p *poly) SqrCached() (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformCached(n)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.Sqr()
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

// SqrCachedWithBump computes p*p using cached transform and bump allocator.
func (p *poly) SqrCachedWithBump(ba *BumpAllocator) (poly, error) {
	n := valueSize(p.k, p.m, 2)

	pv, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		return poly{}, err
	}
	rv, err := pv.SqrWithBump(ba)
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
