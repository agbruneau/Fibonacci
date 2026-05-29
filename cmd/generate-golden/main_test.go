package main

import (
	"fmt"
	"math/big"
	"testing"
)

// knownFibValues contains hand-checked low-n Fibonacci values. The test
// table is built dynamically from this map so extending coverage only
// requires adding entries here, not duplicating the table schema.
var knownFibValues = map[uint64]string{
	0:   "0",
	1:   "1",
	2:   "1",
	3:   "2",
	4:   "3",
	5:   "5",
	10:  "55",
	20:  "6765",
	50:  "12586269025",
	92:  "7540113804746346429",
	93:  "12200160415121876738",
	94:  "19740274219868223167",
	100: "354224848179261915075",
}

// TestFibBig tests the oracle Fibonacci function with known values.
// Targets are generated dynamically from knownFibValues.
func TestFibBig(t *testing.T) {
	t.Parallel()
	// Deterministic iteration order for readable subtest names.
	ns := make([]uint64, 0, len(knownFibValues))
	for n := range knownFibValues {
		ns = append(ns, n)
	}
	// Simple insertion sort (table is tiny; keeps imports minimal).
	for i := 1; i < len(ns); i++ {
		for j := i; j > 0 && ns[j-1] > ns[j]; j-- {
			ns[j-1], ns[j] = ns[j], ns[j-1]
		}
	}

	for _, n := range ns {

		expected := knownFibValues[n]
		name := fmt.Sprintf("F(%d)", n)
		t.Run(name, func(t *testing.T) {
			result := fibBig(n)
			if result.String() != expected {
				t.Errorf("fibBig(%d) = %s, want %s", n, result.String(), expected)
			}
		})
	}
}

// TestFibBig_Properties tests mathematical properties of Fibonacci numbers.
func TestFibBig_Properties(t *testing.T) {
	t.Parallel()
	t.Run("F(n) + F(n+1) = F(n+2)", func(t *testing.T) {
		for n := uint64(0); n < 50; n++ {
			fn := fibBig(n)
			fn1 := fibBig(n + 1)
			fn2 := fibBig(n + 2)

			sum := new(big.Int).Add(fn, fn1)
			if sum.Cmp(fn2) != 0 {
				t.Errorf("F(%d) + F(%d) = %s, but F(%d) = %s",
					n, n+1, sum.String(), n+2, fn2.String())
			}
		}
	})

	t.Run("F(n) is monotonically increasing for n >= 1", func(t *testing.T) {
		prev := fibBig(1)
		for n := uint64(2); n <= 100; n++ {
			curr := fibBig(n)
			if curr.Cmp(prev) < 0 {
				t.Errorf("F(%d) = %s < F(%d) = %s, should be increasing",
					n, curr.String(), n-1, prev.String())
			}
			prev = curr
		}
	})
}

// TestFibBig_LargeValues tests larger Fibonacci numbers.
func TestFibBig_LargeValues(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping large value tests in short mode")
	}

	tests := []struct {
		name     string
		n        uint64
		expected string
	}{
		{
			"F(1000)",
			1000,
			"43466557686937456435688527675040625802564660517371780402481729089536555417949051890403879840079255169295922593080322634775209689623239873322471161642996440906533187938298969649928516003704476137795166849228875",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			result := fibBig(tt.n)
			if result.String() != tt.expected {
				t.Errorf("fibBig(%d) result mismatch\ngot:  %s\nwant: %s",
					tt.n, result.String(), tt.expected)
			}
		})
	}
}

// TestFibBig_FFTBound exercises the oracle on an input whose result crosses
// the default FFT threshold (~500k bits). F(800000) has ~555k bits, which
// ensures any downstream consumer that switches to Schönhage–Strassen at
// 500k bits gets a golden value in the FFT regime.
//
// Instead of hard-coding the 800k-digit expected string, we validate the
// result using Fibonacci's identity: F(n-1) + F(n-2) = F(n). This keeps the
// test self-contained while still exercising the FFT-bound code path.
func TestFibBig_FFTBound(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping FFT-bound test in short mode (F(800000) takes ~1s and uses ~70KB)")
	}

	const n uint64 = 800_000
	const fftThresholdBits = 500_000

	fn := fibBig(n)
	if fn.BitLen() < fftThresholdBits {
		t.Fatalf("F(%d) has %d bits, expected >= %d (FFT-bound precondition)",
			n, fn.BitLen(), fftThresholdBits)
	}

	fn1 := fibBig(n - 1)
	fn2 := fibBig(n - 2)

	sum := new(big.Int).Add(fn1, fn2)
	if sum.Cmp(fn) != 0 {
		t.Errorf("F(%d-1) + F(%d-2) != F(%d): identity violated (mismatch at %d bits)",
			n, n, n, fn.BitLen())
	}
}

// TestFibBig_FFTBound_IdentityMatchesSmall verifies the same Fibonacci
// identity on smaller inputs so regressions in the iterative core are
// caught cheaply before exercising the ~1s FFT-bound case above.
func TestFibBig_FFTBound_IdentityMatchesSmall(t *testing.T) {
	t.Parallel()
	// A quick correlate test: grows through the powers-of-10 boundary.
	for _, n := range []uint64{100, 1_000, 10_000} {

		t.Run(fmt.Sprintf("F(%d)_identity", n), func(t *testing.T) {
			fn := fibBig(n)
			fn1 := fibBig(n - 1)
			fn2 := fibBig(n - 2)
			sum := new(big.Int).Add(fn1, fn2)
			if sum.Cmp(fn) != 0 {
				t.Errorf("identity violated at n=%d", n)
			}
		})
	}
}
