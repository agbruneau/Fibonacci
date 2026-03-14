// This file provides a thread-safe LRU cache for FFT transform results.

package bigfft

import (
	"container/list"
	"encoding/binary"
	"hash/fnv"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

// ─────────────────────────────────────────────────────────────────────────────
// FFT Transform Cache
// ─────────────────────────────────────────────────────────────────────────────

// TransformCacheConfig holds configuration for the FFT transform cache.
type TransformCacheConfig struct {
	// MaxEntries is the maximum number of cached transforms.
	// Default: 256 entries
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
		MaxEntries: 256,
		MinBitLen:  100000,
		Enabled:    true,
	}
}

// cacheEntry holds a cached FFT transform result.
type cacheEntry struct {
	key    uint64   // FNV-1a hash of input
	values []fermat // cached polValues.values
	k      uint     // FFT size parameter
	n      int      // coefficient length
}

// cacheLogInterval is the number of accesses between periodic cache stats logging.
const cacheLogInterval = 100

// TransformCache is a thread-safe LRU cache for FFT transforms.
// It caches the forward FFT transform results to avoid recomputation
// when the same values are multiplied repeatedly.
type TransformCache struct {
	mu        sync.RWMutex
	config    TransformCacheConfig
	entries   map[uint64]*list.Element
	lru       *list.List
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
	accesses  atomic.Uint64
	logger    zerolog.Logger
}

// NewTransformCache creates a new FFT transform cache with the given config.
func NewTransformCache(config TransformCacheConfig) *TransformCache {
	return &TransformCache{
		config:  config,
		entries: make(map[uint64]*list.Element),
		lru:     list.New(),
		logger:  zerolog.Nop(),
	}
}

// SetCacheLogger configures the logger for the global FFT transform cache.
func SetCacheLogger(l zerolog.Logger) {
	cache := GetTransformCache()
	cache.logger = l
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
		cache.entries = make(map[uint64]*list.Element)
		cache.lru.Init()
	}
}

// computeCacheKey generates a cache key from the input data using FNV-1a.
// FNV-1a is much faster than SHA-256 and provides sufficient collision
// resistance for cache key purposes.
func computeCacheKey(data nat, k uint, n int) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(k))
	h.Write(buf[:])
	binary.LittleEndian.PutUint64(buf[:], uint64(n))
	h.Write(buf[:])
	for _, word := range data {
		binary.LittleEndian.PutUint64(buf[:], uint64(word))
		h.Write(buf[:])
	}
	return h.Sum64()
}

// computePolyKey generates a cache key directly from polynomial coefficients,
// avoiding the intermediate allocation of flattenPolyData.
func computePolyKey(p *Poly, k uint, n int) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(k))
	h.Write(buf[:])
	binary.LittleEndian.PutUint64(buf[:], uint64(n))
	h.Write(buf[:])
	for _, a := range p.A {
		for _, word := range a {
			binary.LittleEndian.PutUint64(buf[:], uint64(word))
			h.Write(buf[:])
		}
	}
	return h.Sum64()
}

// computeKey is an alias for computeCacheKey for backward compatibility.
func computeKey(data nat, k uint, n int) uint64 {
	return computeCacheKey(data, k, n)
}

// Get retrieves a cached transform if available.
// Returns the PolValues and true if found, zero values and false otherwise.
// IMPORTANT: The returned PolValues references internal cache data and MUST NOT
// be modified. PolValues.Mul() and PolValues.Sqr() are safe as they create new
// result values without mutating the receiver.
func (tc *TransformCache) Get(data nat, k uint, n int) (PolValues, bool) {
	if !tc.config.Enabled || len(data)*_W < tc.config.MinBitLen {
		return PolValues{}, false
	}

	key := computeCacheKey(data, k, n)

	return tc.getByKey(key)
}

// getByKey retrieves a cached transform by precomputed key.
// The returned PolValues shares its backing data with the cache.
// Callers MUST NOT modify the returned values. PolValues.Mul() and
// PolValues.Sqr() are safe as they produce new result values.
func (tc *TransformCache) getByKey(key uint64) (PolValues, bool) {
	tc.mu.RLock()
	elem, found := tc.entries[key]
	tc.mu.RUnlock()

	if !found {
		tc.misses.Add(1)
		tc.logPeriodicStats()
		return PolValues{}, false
	}

	tc.mu.Lock()
	tc.lru.MoveToFront(elem)
	tc.mu.Unlock()

	tc.hits.Add(1)
	tc.logPeriodicStats()

	entry := elem.Value.(*cacheEntry)

	// Return a reference to cached values (zero-copy).
	// Safe because Mul/Sqr create new result slices without mutating inputs.
	return PolValues{
		K:      entry.k,
		N:      entry.n,
		Values: entry.values,
	}, true
}

// logPeriodicStats logs cache statistics every cacheLogInterval accesses.
func (tc *TransformCache) logPeriodicStats() {
	count := tc.accesses.Add(1)
	if count%cacheLogInterval != 0 {
		return
	}
	hits := tc.hits.Load()
	misses := tc.misses.Load()
	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	tc.mu.RLock()
	size := tc.lru.Len()
	tc.mu.RUnlock()
	tc.logger.Debug().
		Uint64("hits", hits).
		Uint64("misses", misses).
		Float64("hit_rate", hitRate).
		Int("size", size).
		Uint64("evictions", tc.evictions.Load()).
		Msg("fft cache stats")
}

// Put stores a transform result in the cache.
func (tc *TransformCache) Put(data nat, pv PolValues) {
	if !tc.config.Enabled || len(data)*_W < tc.config.MinBitLen {
		return
	}

	key := computeCacheKey(data, pv.K, pv.N)

	tc.putByKey(key, pv)
}

// putByKey stores a transform result in the cache by precomputed key.
func (tc *TransformCache) putByKey(key uint64, pv PolValues) {
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

	// Create a deep copy using a contiguous buffer to reduce allocations
	K := len(pv.Values)
	n := pv.N
	wordCount := K * (n + 1)
	backing := make([]big.Word, wordCount)
	valuesCopy := make([]fermat, K)
	for i, v := range pv.Values {
		valuesCopy[i] = fermat(backing[i*(n+1) : (i+1)*(n+1)])
		copy(valuesCopy[i], v)
	}

	entry := &cacheEntry{
		key:    key,
		values: valuesCopy,
		k:      pv.K,
		n:      pv.N,
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

	tc.entries = make(map[uint64]*list.Element)
	tc.lru.Init()
	tc.hits.Store(0)
	tc.misses.Store(0)
	tc.evictions.Store(0)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cached Transform Functions
// ─────────────────────────────────────────────────────────────────────────────

// polyBitLen estimates the total bit length of polynomial coefficients.
func polyBitLen(p *Poly) int {
	totalWords := 0
	for _, a := range p.A {
		totalWords += len(a)
	}
	return totalWords * _W
}

// TransformCached is like Transform but uses the global cache.
// If the transform result is cached, it returns the cached value.
// Otherwise, it computes the transform and caches the result.
func (p *Poly) TransformCached(n int) (PolValues, error) {
	cache := GetTransformCache()

	// Check if caching is applicable
	if !cache.config.Enabled || polyBitLen(p) < cache.config.MinBitLen {
		return p.Transform(n)
	}

	// Compute key directly from polynomial coefficients (no intermediate allocation)
	key := computePolyKey(p, p.K, n)

	// Try cache lookup
	if cached, found := cache.getByKey(key); found {
		return cached, nil
	}

	// Compute transform
	pv, err := p.Transform(n)
	if err != nil {
		return PolValues{}, err
	}

	// Cache the result
	cache.putByKey(key, pv)

	return pv, nil
}

// TransformCachedWithBump is like TransformWithBump but uses the global cache.
func (p *Poly) TransformCachedWithBump(n int, ba *BumpAllocator) (PolValues, error) {
	cache := GetTransformCache()

	// Check if caching is applicable
	if !cache.config.Enabled || polyBitLen(p) < cache.config.MinBitLen {
		return p.TransformWithBump(n, ba)
	}

	// Compute key directly from polynomial coefficients (no intermediate allocation)
	key := computePolyKey(p, p.K, n)

	// Try cache lookup
	if cached, found := cache.getByKey(key); found {
		return cached, nil
	}

	// Compute transform
	pv, err := p.TransformWithBump(n, ba)
	if err != nil {
		return PolValues{}, err
	}

	// Cache the result
	cache.putByKey(key, pv)

	return pv, nil
}

// MulCached multiplies p and q using cached transforms when beneficial.
func (p *Poly) MulCached(q *Poly) (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := p.TransformCached(n)
	if err != nil {
		return Poly{}, err
	}
	qv, err := q.TransformCached(n)
	if err != nil {
		return Poly{}, err
	}
	rv, err := pv.Mul(&qv)
	if err != nil {
		return Poly{}, err
	}
	r, err := rv.InvTransform()
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}

// MulCachedWithBump multiplies p and q using cached transforms and bump allocator.
func (p *Poly) MulCachedWithBump(q *Poly, ba *BumpAllocator) (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		return Poly{}, err
	}
	qv, err := q.TransformCachedWithBump(n, ba)
	if err != nil {
		return Poly{}, err
	}
	rv, err := pv.MulWithBump(&qv, ba)
	if err != nil {
		return Poly{}, err
	}
	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}

// SqrCached computes p*p using cached transform when beneficial.
func (p *Poly) SqrCached() (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := p.TransformCached(n)
	if err != nil {
		return Poly{}, err
	}
	rv, err := pv.Sqr()
	if err != nil {
		return Poly{}, err
	}
	r, err := rv.InvTransform()
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}

// SqrCachedWithBump computes p*p using cached transform and bump allocator.
func (p *Poly) SqrCachedWithBump(ba *BumpAllocator) (Poly, error) {
	n := valueSize(p.K, p.M, 2)

	pv, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		return Poly{}, err
	}
	rv, err := pv.SqrWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}
