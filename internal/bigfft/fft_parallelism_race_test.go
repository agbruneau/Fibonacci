package bigfft

import (
	"math/big"
	"sync"
	"testing"
	"time"
)

// TestFFTParallelismConfigRace is the [A-03] regression test.
//
// fftThreshold (fft.go), ParallelFFTRecursionThreshold and
// MaxParallelFFTDepth (fft_recursion.go) are read at every FFT recursion
// node by N parallel goroutines while SetFFTParallelismConfig mutates the
// latter two with a plain assignment — an unsynchronized read/write data
// race.
//
// This test runs many concurrent large-int FFT multiplications (which dive
// through fourierRecursiveUnified and read the parallelism globals on every
// node) while a second goroutine continuously calls
// SetFFTParallelismConfig. Under `go test -race` (CI) this deterministically
// trips the race detector on the threshold globals. In normal mode it
// asserts the results stay correct and no panic occurs under the
// reconfiguration storm (a torn uint read is unlikely to be visible on
// amd64, so the value check is a weak guard; -race is the real oracle).
func TestFFTParallelismConfigRace(t *testing.T) {
	t.Parallel()

	// Restore the configuration after the test so other tests/benchmarks
	// see the documented defaults.
	orig := GetFFTParallelismConfig()
	t.Cleanup(func() { SetFFTParallelismConfig(orig) })

	// Operands large enough (> fftThreshold words) to force the FFT path.
	x := makeBigInt(11, 4000)
	y := makeBigInt(22, 4000)
	want := new(big.Int).Mul(x, y)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var mismatch sync.Once
	var failed bool

	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				got, err := Mul(x, y)
				if err != nil {
					mismatch.Do(func() { failed = true })
					return
				}
				if got.Cmp(want) != 0 {
					mismatch.Do(func() { failed = true })
					return
				}
			}
		}()
	}

	// Mutator: hammer the parallelism config under the readers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		flip := false
		for {
			select {
			case <-stop:
				return
			default:
			}
			flip = !flip
			if flip {
				SetFFTParallelismConfig(FFTParallelismConfig{RecursionThreshold: 3, MaxDepth: 2})
			} else {
				SetFFTParallelismConfig(FFTParallelismConfig{RecursionThreshold: 5, MaxDepth: 4})
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()

	if failed {
		t.Fatal("[A-03] FFT result mismatch / error under concurrent " +
			"SetFFTParallelismConfig")
	}
}
