// This file provides a thread-safe LRU cache for FFT transform results.

package bigfft

import (
	"container/list"
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
	key     uint64     // FNV-1a hash of input
	values  []fermat   // cached polValues.values (sub-slices of backing)
	backing []big.Word // contiguous storage for values; recycled on eviction
	k       uint       // FFT size parameter
	n       int        // coefficient length

	// refs counts live consumers that obtained a PolValues aliasing this
	// entry's backing via getByKey but have not yet Release()d it. Eviction
	// (putByKey) MUST NOT salvage/zero `backing` while refs > 0, otherwise a
	// concurrent reader observes silent corruption (audit A-01). When refs
	// is non-zero on eviction the backing is dropped to GC instead of being
	// recycled; this is rare (only under simultaneous churn + in-flight FFT)
	// and trades a transient allocation for memory safety.
	refs atomic.Int32
}

// cacheLogInterval is the number of accesses between periodic cache stats logging.
const cacheLogInterval = 100

// TransformCache is a thread-safe LRU cache for FFT transforms.
// It caches the forward FFT transform results to avoid recomputation
// when the same values are multiplied repeatedly.
type TransformCache struct {
	mu sync.RWMutex
	// Config is stored as atomics rather than a plain struct because Get/Put
	// read it on the lock-free fast path while SetConfig mutates it under
	// mu.Lock() — a plain field is a data race (audit A-02). The public
	// TransformCacheConfig type keeps value fields; conversion happens at the
	// loadConfig/storeConfig boundary. These are existing-state replacements,
	// not new package globals.
	cfgEnabled    atomic.Bool
	cfgMinBitLen  atomic.Int64
	cfgMaxEntries atomic.Int64
	entries       map[uint64]*list.Element
	lru           *list.List
	hits          atomic.Uint64
	misses        atomic.Uint64
	evictions     atomic.Uint64
	accesses      atomic.Uint64
	logger        zerolog.Logger
}

// loadConfig reconstructs the public config value from the atomic fields.
// Lock-free: safe to call on the Get/Put hot path.
func (tc *TransformCache) loadConfig() TransformCacheConfig {
	return TransformCacheConfig{
		MaxEntries: int(tc.cfgMaxEntries.Load()),
		MinBitLen:  int(tc.cfgMinBitLen.Load()),
		Enabled:    tc.cfgEnabled.Load(),
	}
}

// storeConfig writes the public config value into the atomic fields.
func (tc *TransformCache) storeConfig(c TransformCacheConfig) {
	tc.cfgMaxEntries.Store(int64(c.MaxEntries))
	tc.cfgMinBitLen.Store(int64(c.MinBitLen))
	tc.cfgEnabled.Store(c.Enabled)
}

// Config returns the current configuration of the transform cache.
func (tc *TransformCache) Config() TransformCacheConfig {
	return tc.loadConfig()
}

// SetConfig atomically updates this cache's configuration. When the new
// config disables the cache, existing entries are cleared under mu.Lock()
// (the map/list mutation still requires the exclusive lock); the config
// atomics themselves are lock-free for readers.
func (tc *TransformCache) SetConfig(config TransformCacheConfig) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.storeConfig(config)
	if !config.Enabled {
		tc.entries = make(map[uint64]*list.Element)
		tc.lru.Init()
	}
}

// NewTransformCache creates a new FFT transform cache with the given config.
func NewTransformCache(config TransformCacheConfig) *TransformCache {
	tc := &TransformCache{
		entries: make(map[uint64]*list.Element),
		lru:     list.New(),
		logger:  zerolog.Nop(),
	}
	tc.storeConfig(config)
	return tc
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

// FNV-1a constants for 64-bit hash.
const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211
)

// SetTransformCacheConfig updates the global cache configuration.
// This should be called before any FFT operations for consistent behavior.
// It delegates to the per-instance SetConfig so the atomic write/clear
// semantics are shared with non-global caches (audit A-02).
func SetTransformCacheConfig(config TransformCacheConfig) {
	GetTransformCache().SetConfig(config)
}

// cacheKeyBuilder accumulates an FNV-1a 64-bit hash. It is the single
// implementation behind every cache-key computation in this file; both
// computeCacheKey and computePolyKey use it to ensure bit-identical hashes
// across input shapes (nat slice vs. polynomial coefficients).
type cacheKeyBuilder struct {
	h uint64
}

// newCacheKeyBuilder returns a builder initialized with the FNV-1a offset.
func newCacheKeyBuilder() cacheKeyBuilder {
	return cacheKeyBuilder{h: offset64}
}

// writeUint64 mixes a uint64 into the hash, byte by byte in little-endian
// order. This matches the historical fnvWriteUint64 byte-ordering exactly so
// that previously persisted/expected keys remain valid.
func (b *cacheKeyBuilder) writeUint64(x uint64) {
	h := b.h
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
	b.h = h
}

// writeNat mixes every word of a nat into the hash in slice order.
func (b *cacheKeyBuilder) writeNat(data nat) {
	for _, word := range data {
		b.writeUint64(uint64(word))
	}
}

// sum returns the accumulated 64-bit hash.
func (b *cacheKeyBuilder) sum() uint64 { return b.h }

// computeCacheKey generates a cache key from the input data using FNV-1a.
// FNV-1a is much faster than SHA-256 and provides sufficient collision
// resistance for cache key purposes.
func computeCacheKey(data nat, k uint, n int) uint64 {
	b := newCacheKeyBuilder()
	b.writeUint64(uint64(k))
	b.writeUint64(uint64(n)) // #nosec G115 -- bit-pattern reinterpretation for FNV hash; n is a non-negative size
	b.writeNat(data)
	return b.sum()
}

// computePolyKey generates a cache key directly from polynomial coefficients,
// avoiding the intermediate allocation of flattenPolyData.
func computePolyKey(p *Poly, k uint, n int) uint64 {
	b := newCacheKeyBuilder()
	b.writeUint64(uint64(k))
	b.writeUint64(uint64(n)) // #nosec G115 -- bit-pattern reinterpretation for FNV hash; n is a non-negative size
	for _, a := range p.A {
		b.writeNat(a)
	}
	return b.sum()
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
	if !tc.cfgEnabled.Load() || int64(len(data)*_W) < tc.cfgMinBitLen.Load() {
		return PolValues{}, false
	}

	key := computeCacheKey(data, k, n)

	return tc.getByKey(key)
}

// getByKey retrieves a cached transform by precomputed key.
// The returned PolValues shares its backing data with the cache.
// Callers MUST NOT modify the returned values. PolValues.Mul() and
// PolValues.Sqr() are safe as they produce new result values.
//
// Hot-path optimization (R3.9): under a single RLock we both look up the
// entry and snapshot its fields. We then check whether the element is
// already at the LRU front; if so, no exclusive lock is taken (a
// MoveToFront on the front is a documented no-op, so LRU order is strictly
// preserved). Otherwise we acquire a brief Lock for MoveToFront. On a very
// hot cache (entries that stay near the front), this eliminates most
// exclusive Lock acquisitions, removing the contention measured at 10k+
// accesses/s.
//
// Lifetime safety (audit A-01): the entry's k/n are immutable, but its
// `backing` (and thus the `values` sub-slices we hand back) is NOT — under
// eviction putByKey may salvage and zero it. We therefore pin the entry by
// incrementing its atomic refcount while still holding the RLock, before
// any concurrent putByKey can observe the entry as evictable. The returned
// PolValues carries cacheRef so its Release() drops the pin; putByKey only
// recycles a backing whose refs == 0.
func (tc *TransformCache) getByKey(key uint64) (PolValues, bool) {
	tc.mu.RLock()
	elem, found := tc.entries[key]
	if !found {
		tc.mu.RUnlock()
		tc.misses.Add(1)
		tc.logPeriodicStats()
		return PolValues{}, false
	}

	// Pin the entry under RLock so a concurrent putByKey eviction cannot
	// salvage/zero this backing while the caller reads the returned values.
	entry := elem.Value.(*cacheEntry)
	entry.refs.Add(1)
	pv := PolValues{
		K:        entry.k,
		N:        entry.n,
		Values:   entry.values,
		cacheRef: entry,
	}
	// Fast path: if the element is already at the front, MoveToFront would
	// be a no-op. Skip the exclusive Lock entirely. Reading lru.Front() is
	// safe under RLock (no concurrent mutators).
	atFront := tc.lru.Front() == elem
	tc.mu.RUnlock()

	if !atFront {
		tc.mu.Lock()
		// Re-check membership: a concurrent eviction between RUnlock and
		// Lock would have removed elem from the list. MoveToFront is
		// documented as a no-op when the element is not in the list, so
		// this is safe — but the explicit guard avoids any future
		// container/list semantic drift and documents the intent.
		if _, stillPresent := tc.entries[key]; stillPresent {
			tc.lru.MoveToFront(elem)
		}
		tc.mu.Unlock()
	}

	tc.hits.Add(1)
	tc.logPeriodicStats()

	return pv, true
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
//
// Invariant (audit A-05): every coefficient of pv must have exactly
// pv.N+1 words. A PolValues produced by the FFT transform pipeline always
// satisfies this. If any coefficient violates it, the whole pv is dropped
// (not cached) rather than copied: a short coefficient would be silently
// truncated by the internal copy and a later cache hit would return a
// corrupt transform. Dropping turns a malformed Put into a clean future
// miss (recompute), never into data corruption.
func (tc *TransformCache) Put(data nat, pv PolValues) {
	if !tc.cfgEnabled.Load() || int64(len(data)*_W) < tc.cfgMinBitLen.Load() {
		return
	}

	key := computeCacheKey(data, pv.K, pv.N)

	tc.putByKey(key, pv)
}

// putByKey stores a transform result in the cache by precomputed key.
//
// When the cache is at capacity, the oldest entry is evicted and its
// contiguous backing buffer is recycled to back the new entry whenever its
// capacity is sufficient AND no getByKey consumer still pins it (audit
// A-01: a pinned backing is dropped to GC instead, never zeroed in place).
// This avoids an O(K·n) allocation on the hot path of cache turnover.
// Eviction order (LRU) and thread-safety are preserved.
func (tc *TransformCache) putByKey(key uint64, pv PolValues) {
	n := pv.N
	// A-05: reject malformed shapes BEFORE taking the lock or touching the
	// LRU. Caching a pv whose coefficients are not all exactly n+1 words
	// would let the internal copy truncate them, corrupting future hits.
	// Drop instead — a clean miss is always recoverable, corruption is not.
	for i := range pv.Values {
		if len(pv.Values[i]) != n+1 {
			return
		}
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Check if already cached
	if _, found := tc.entries[key]; found {
		return
	}

	K := len(pv.Values)
	wordCount := K * (n + 1)

	// Evict oldest entries if at capacity, salvaging a backing buffer
	// large enough to host the new entry when possible.
	var backing []big.Word
	maxEntries := int(tc.cfgMaxEntries.Load())
	for tc.lru.Len() >= maxEntries {
		oldest := tc.lru.Back()
		if oldest == nil {
			break
		}
		tc.lru.Remove(oldest)
		entry := oldest.Value.(*cacheEntry)
		delete(tc.entries, entry.key)
		tc.evictions.Add(1)

		// Salvage the largest suitable evicted backing. We keep the first
		// fit (LRU order); subsequent evictions in the same loop drop their
		// buffers to GC, matching the prior semantics for them.
		//
		// A-01: never recycle a backing that a getByKey consumer is still
		// reading (refs > 0). Doing so would zero/overwrite live data under
		// an in-flight FFT. Such a backing is dropped to GC; the in-flight
		// reader keeps a valid (now private) view until it Release()s.
		if backing == nil && entry.refs.Load() == 0 && cap(entry.backing) >= wordCount {
			backing = entry.backing[:wordCount]
			// Zero recycled storage to avoid leaking stale words into the
			// new entry's fermat sub-slices (copy below only writes len(v)
			// words per coefficient, which may be < n+1).
			for i := range backing {
				backing[i] = 0
			}
		}
	}

	// Allocate fresh storage if no suitable buffer was salvaged.
	if backing == nil {
		backing = make([]big.Word, wordCount)
	}

	valuesCopy := make([]fermat, K)
	for i, v := range pv.Values {
		valuesCopy[i] = fermat(backing[i*(n+1) : (i+1)*(n+1)])
		copy(valuesCopy[i], v)
	}

	entry := &cacheEntry{
		key:     key,
		values:  valuesCopy,
		backing: backing,
		k:       pv.K,
		n:       pv.N,
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
	if !cache.cfgEnabled.Load() || int64(polyBitLen(p)) < cache.cfgMinBitLen.Load() {
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
	if !cache.cfgEnabled.Load() || int64(polyBitLen(p)) < cache.cfgMinBitLen.Load() {
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
	// Release pv/qv pooled backing (no-op on cache hits because the cache
	// stores its own, non-pooled copy).
	defer pv.Release()

	qv, err := q.TransformCached(n)
	if err != nil {
		return Poly{}, err
	}
	defer qv.Release()

	rv, err := pv.Mul(&qv)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

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
	// Release pv/qv pooled backing (no-op on cache hits; the bump-allocator
	// path still uses sync.Pool for the returned PolValues storage — see
	// transform() — so Release is meaningful here too).
	defer pv.Release()

	qv, err := q.TransformCachedWithBump(n, ba)
	if err != nil {
		return Poly{}, err
	}
	defer qv.Release()

	rv, err := pv.MulWithBump(&qv, ba)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

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
	defer pv.Release()

	rv, err := pv.Sqr()
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

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
	defer pv.Release()

	rv, err := pv.SqrWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	defer rv.Release()

	r, err := rv.InvTransformWithBump(ba)
	if err != nil {
		return Poly{}, err
	}
	r.M = p.M
	return r, nil
}
