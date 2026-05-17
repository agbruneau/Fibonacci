package bigfft

import (
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConfigConcurrentAccess is the [A-02] regression test.
//
// Get/Put (and putByKey) read tc.config.{Enabled,MinBitLen,MaxEntries}
// with no synchronization while SetTransformCacheConfig mutates the same
// fields under tc.mu.Lock(). That is a textbook data race.
//
// Unlike A-01 this corruption is not observable as a value flip in normal
// mode (a torn read of an int/bool on amd64 is unlikely to be visible),
// so this test is primarily a -race CI tripwire: under `go test -race`
// it deterministically reports the concurrent read/write on the config
// fields. In normal mode it only asserts the cache stays functional and
// does not panic under the concurrent reconfiguration storm.
func TestConfigConcurrentAccess(t *testing.T) {
	t.Parallel()

	cache := NewTransformCache(TransformCacheConfig{
		MaxEntries: 32,
		MinBitLen:  64,
		Enabled:    true,
	})

	data := make(nat, 64)
	for i := range data {
		data[i] = big.Word(i + 1)
	}
	vals := PolValues{K: 4, N: 4, Values: []fermat{make(fermat, 5), make(fermat, 5)}}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers: Get/Put hammer the unsynchronized config reads.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			d := make(nat, 64)
			d[0] = big.Word(0xC000 + id)
			for {
				select {
				case <-stop:
					return
				default:
				}
				cache.Put(d, vals)
				cache.Get(data, 4, 4)
			}
		}(r)
	}

	// Writer: continuously reconfigures under Lock (the racing mutator).
	wg.Add(1)
	go func() {
		defer wg.Done()
		toggle := false
		for {
			select {
			case <-stop:
				return
			default:
			}
			toggle = !toggle
			cache.SetConfig(TransformCacheConfig{
				MaxEntries: 16 + boolToInt(toggle)*16,
				MinBitLen:  64,
				Enabled:    true,
			})
		}
	}()

	time.Sleep(120 * time.Millisecond)
	close(stop)
	wg.Wait()

	// Functional sanity: cache still answers and config is coherent.
	got := cache.Config()
	if got.MinBitLen != 64 {
		t.Fatalf("[A-02] config corrupted under concurrency: MinBitLen=%d", got.MinBitLen)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// TestGetByKeyBackingNotMutatedWhilePinned is the [A-01] regression test.
//
// getByKey returns a PolValues whose Values sub-slices alias the cache
// entry's contiguous `backing`. Under eviction, putByKey used to salvage
// that exact backing, zero it, and re-fill it for a new entry — even when a
// concurrent reader was still holding the returned PolValues mid-FFT. That
// is a silent use-after-free / data corruption.
//
// The fix pins the entry via an atomic refcount taken under RLock in
// getByKey and released by PolValues.Release(); putByKey only salvages a
// backing whose refs == 0.
//
// This test pins one PolValues, snapshots its words, then hammers Put with
// fresh keys to force eviction churn. The pinned snapshot must never change
// for the lifetime of the pin. Under the buggy code, the salvage+zero of
// the pinned backing flips the words to zero and the comparison fails.
//
// NOTE: this reproduces deterministically even WITHOUT the race detector
// because the corruption is an observable value change, not just a racy
// access. Under `-race` (CI) it additionally flags the unsynchronized
// read/write on the shared backing.
func TestGetByKeyBackingNotMutatedWhilePinned(t *testing.T) {
	t.Parallel()

	const (
		maxEntries = 4
		K          = 8
		N          = 32 // each fermat coefficient has length N+1
	)
	config := TransformCacheConfig{
		MaxEntries: maxEntries,
		MinBitLen:  64,
		Enabled:    true,
	}
	cache := NewTransformCache(config)

	makeValues := func(seed int) PolValues {
		vals := make([]fermat, K)
		for i := range vals {
			vals[i] = make(fermat, N+1)
			for j := range vals[i] {
				vals[i][j] = big.Word(0x5A5A0000 + seed*1000 + i*10 + j)
			}
		}
		return PolValues{K: 4, N: N, Values: vals}
	}

	// Seed the pinned entry with a distinctive key/payload.
	pinnedData := make(nat, 10)
	pinnedData[0] = big.Word(0xDEADBEEF)
	cache.Put(pinnedData, makeValues(1))

	// Pin it via getByKey (the consumer path under audit).
	key := computeCacheKey(pinnedData, 4, N)
	pv, found := cache.getByKey(key)
	if !found {
		t.Fatal("expected pinned entry to be present")
	}

	// Snapshot the pinned words. After this, while we hold pv (and have NOT
	// called Release), no Put on the cache may mutate these words.
	snapshot := make([][]big.Word, len(pv.Values))
	for i := range pv.Values {
		snapshot[i] = append([]big.Word(nil), pv.Values[i]...)
	}

	var corrupt atomic.Bool
	var wg sync.WaitGroup

	// Reader: continuously verifies the pinned values match the snapshot.
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			for i := range pv.Values {
				for j := range pv.Values[i] {
					if pv.Values[i][j] != snapshot[i][j] {
						corrupt.Store(true)
						return
					}
				}
			}
		}
	}()

	// Writers: saturate the cache with fresh keys to force constant eviction.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < 4000; i++ {
				d := make(nat, 10)
				d[0] = big.Word(0xB0000000 + base*100000 + i)
				cache.Put(d, makeValues(base*100000+i+2))
			}
		}(w)
	}

	// Let the churn run briefly.
	time.Sleep(150 * time.Millisecond)
	close(stop)
	wg.Wait()

	if corrupt.Load() {
		t.Fatal("[A-01] pinned PolValues backing was mutated by a concurrent " +
			"Put eviction (use-after-free / silent corruption)")
	}

	// Releasing the pin must let the entry's backing become salvageable
	// again (no leak): after Release, refs drops to 0.
	pv.Release()
}
