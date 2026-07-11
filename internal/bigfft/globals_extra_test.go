// Tests in this file mutate package-level state (FFT threshold, parallelism
// knobs, the global transform cache, the global concurrency semaphore, the
// pool-warming flag). Per the global-state test policy (incident fixed in
// commit 9cad06e): none of them calls t.Parallel(), and each restores the
// initial state via t.Cleanup so the parallel test phase observes defaults.

package bigfft

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// makeBigInt builds a big.Int with the given word count and a deterministic
// pattern so we can exercise the FFT path (>fftThreshold words).
func makeBigInt(seed, numWords int) *big.Int {
	z := new(big.Int)
	words := make([]big.Word, numWords)
	for i := range words {
		words[i] = big.Word(uint64(seed)*0x9E3779B97F4A7C15 + uint64(i))
	}
	z.SetBits(words)
	return z
}

// TestSetFFTThresholdRoundTrip verifies the atomic threshold setter/getter
// pair and that a lowered threshold still yields correct products through
// the FFT route.
func TestSetFFTThresholdRoundTrip(t *testing.T) {
	orig := getFFTThreshold()
	t.Cleanup(func() { SetFFTThreshold(orig) })

	SetFFTThreshold(10)
	if got := getFFTThreshold(); got != 10 {
		t.Fatalf("getFFTThreshold() = %d after SetFFTThreshold(10)", got)
	}

	// With the threshold lowered, 64-word operands take the FFT path; the
	// result must match math/big regardless of the route taken.
	x := makeBigInt(61, 64)
	y := makeBigInt(62, 64)
	expected := new(big.Int).Mul(x, y)
	got, err := Mul(x, y)
	if err != nil {
		t.Fatalf("Mul under lowered threshold failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatal("Mul under lowered threshold produced a wrong product")
	}
}

// TestSetFFTParallelismConfigRoundTrip verifies that positive fields are
// applied and zero fields leave the current values untouched.
func TestSetFFTParallelismConfigRoundTrip(t *testing.T) {
	orig := GetFFTParallelismConfig()
	t.Cleanup(func() { SetFFTParallelismConfig(orig) })

	SetFFTParallelismConfig(FFTParallelismConfig{RecursionThreshold: 6, MaxDepth: 5})
	got := GetFFTParallelismConfig()
	if got.RecursionThreshold != 6 || got.MaxDepth != 5 {
		t.Fatalf("config not applied: %+v", got)
	}

	SetFFTParallelismConfig(FFTParallelismConfig{})
	got = GetFFTParallelismConfig()
	if got.RecursionThreshold != 6 || got.MaxDepth != 5 {
		t.Fatalf("zero-valued config must not overwrite current values, got %+v", got)
	}
}

// TestSetCacheLoggerWiresPeriodicStats installs a buffer-backed logger on the
// global cache and drives logPeriodicStats until the periodic emission fires
// (exactly one of cacheLogInterval consecutive accesses hits the modulo).
func TestSetCacheLoggerWiresPeriodicStats(t *testing.T) {
	t.Cleanup(func() { setCacheLogger(zerolog.Nop()) })

	var buf bytes.Buffer
	setCacheLogger(zerolog.New(&buf))

	tc := GetTransformCache()
	for i := 0; i < cacheLogInterval && buf.Len() == 0; i++ {
		tc.logPeriodicStats()
	}
	if !strings.Contains(buf.String(), "fft cache stats") {
		t.Fatalf("periodic stats not emitted through the configured logger: %q", buf.String())
	}
}

// globalCachePoly builds a poly large enough (>= default MinBitLen) to
// qualify for the global transform cache, plus the matching transform length.
func globalCachePoly(seed int) (p Poly, n int) {
	x := makeBigInt(seed, 2000) // 2000 words = 128k bits >= 100k MinBitLen
	var xb nat = x.Bits()
	k, m := fftSize(xb, xb)
	return polyFromNat(xb, k, m), valueSize(k, m, 2)
}

// assertSamePolValues fails when the two transforms differ in shape or words.
func assertSamePolValues(t *testing.T, got, want *PolValues) {
	t.Helper()
	if len(got.Values) != len(want.Values) {
		t.Fatalf("value count mismatch: got %d, want %d", len(got.Values), len(want.Values))
	}
	for i := range want.Values {
		if len(got.Values[i]) != len(want.Values[i]) {
			t.Fatalf("value %d length mismatch: got %d, want %d", i, len(got.Values[i]), len(want.Values[i]))
		}
		for j := range want.Values[i] {
			if got.Values[i][j] != want.Values[i][j] {
				t.Fatalf("cached transform differs at value %d word %d", i, j)
			}
		}
	}
}

// TestTransformCachedGlobalMissThenHit drives (*Poly).TransformCached against
// the global cache: the first call must compute and store (miss), the second
// must be served from the cache (hit) with values identical to an uncached
// reference transform.
func TestTransformCachedGlobalMissThenHit(t *testing.T) {
	orig := GetTransformCache().Config()
	t.Cleanup(func() { SetTransformCacheConfig(orig) })
	SetTransformCacheConfig(DefaultTransformCacheConfig())
	GetTransformCache().Clear() // deterministic absolute stats below

	p, n := globalCachePoly(71)

	ref, err := p.Transform(n)
	if err != nil {
		t.Fatalf("reference Transform failed: %v", err)
	}
	defer ref.Release()

	pv1, err := p.TransformCached(n)
	if err != nil {
		t.Fatalf("first TransformCached failed: %v", err)
	}
	assertSamePolValues(t, &pv1, &ref)
	pv1.Release() // the cache stores its own copy; pooled backing goes back

	pv2, err := p.TransformCached(n)
	if err != nil {
		t.Fatalf("second TransformCached failed: %v", err)
	}
	assertSamePolValues(t, &pv2, &ref)

	stats := GetTransformCache().Stats()
	if stats.Misses != 1 || stats.Hits != 1 {
		t.Errorf("expected exactly 1 miss and 1 hit, got %+v", stats)
	}
	// Cache-shared values have no pooled backing; Release must be a no-op.
	pv2.Release()
}

// TestTransformCachedWithBumpGlobalMissThenHit is the bump-allocator twin of
// the test above.
func TestTransformCachedWithBumpGlobalMissThenHit(t *testing.T) {
	orig := GetTransformCache().Config()
	t.Cleanup(func() { SetTransformCacheConfig(orig) })
	SetTransformCacheConfig(DefaultTransformCacheConfig())
	GetTransformCache().Clear()

	p, n := globalCachePoly(72)

	ba := AcquireBumpAllocator(EstimateBumpCapacity(4000))
	defer ReleaseBumpAllocator(ba)

	ref, err := p.TransformWithBump(n, ba)
	if err != nil {
		t.Fatalf("reference TransformWithBump failed: %v", err)
	}
	defer ref.Release()

	pv1, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		t.Fatalf("first TransformCachedWithBump failed: %v", err)
	}
	assertSamePolValues(t, &pv1, &ref)
	pv1.Release()

	pv2, err := p.TransformCachedWithBump(n, ba)
	if err != nil {
		t.Fatalf("second TransformCachedWithBump failed: %v", err)
	}
	assertSamePolValues(t, &pv2, &ref)
	pv2.Release()

	stats := GetTransformCache().Stats()
	if stats.Misses != 1 || stats.Hits != 1 {
		t.Errorf("expected exactly 1 miss and 1 hit, got %+v", stats)
	}
}

// TestEnsurePoolsWarmedFlag verifies the warm-once contract on the package
// flag: the first call flips it, subsequent calls are no-ops.
func TestEnsurePoolsWarmedFlag(t *testing.T) {
	orig := poolsWarmed.Load()
	t.Cleanup(func() { poolsWarmed.Store(orig) })

	poolsWarmed.Store(false)
	EnsurePoolsWarmed(1000)
	if !poolsWarmed.Load() {
		t.Fatal("EnsurePoolsWarmed must mark pools as warmed")
	}
	EnsurePoolsWarmed(1000) // second call must be a no-op and not panic
	if !poolsWarmed.Load() {
		t.Fatal("warmed flag must stay set after a redundant call")
	}
}

// TestGlobalSemaphoreExhaustedFallsBackSequential fills the package-level FFT
// semaphore so every non-blocking acquire in the recursion fails: the FFT
// must complete sequentially with a correct result (the non-blocking-acquire
// contract documented in fourierRecursiveUnified). Tokens are drained in
// Cleanup so later tests regain parallelism.
func TestGlobalSemaphoreExhaustedFallsBackSequential(t *testing.T) {
	sem := getSemaphore()
	filled := 0
	for filling := true; filling; {
		select {
		case sem <- struct{}{}:
			filled++
		default:
			filling = false
		}
	}
	t.Cleanup(func() {
		for i := 0; i < filled; i++ {
			<-sem
		}
	})

	x := makeBigInt(63, 2000)
	y := makeBigInt(64, 2000)
	expected := new(big.Int).Mul(x, y)
	got, err := Mul(x, y)
	if err != nil {
		t.Fatalf("Mul with exhausted semaphore failed: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatal("sequential fallback produced a wrong product")
	}
}
