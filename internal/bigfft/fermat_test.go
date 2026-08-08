package bigfft

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
)

// TestFermatSqrVsMul verifies that fermat.Sqr(x) produces the same result
// as fermat.Mul(x, x) for various sizes, crossing the smallMulThreshold boundary.
func TestFermatSqrVsMul(t *testing.T) {
	t.Parallel()

	// Test sizes spanning below and above smallMulThreshold (30)
	sizes := []int{1, 2, 3, 5, 10, 15, 20, 25, 29, 30, 31, 35, 40, 50}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			// Per-subtest source: *rand.Rand is not safe for concurrent
			// use, and the subtests run in parallel (data race flagged by
			// -race on the previously shared parent rng).
			rng := rand.New(rand.NewSource(42 + int64(n)))
			// Create random fermat number of size n+1 words
			x := make(fermat, n+1)
			for j := 0; j < n; j++ {
				x[j] = big.Word(rng.Uint64())
			}
			x[n] = big.Word(rng.Intn(2)) // last word 0 or 1

			// Compute via Mul(x, x)
			bufMul := make(fermat, 8*n)
			resMul := bufMul.Mul(x, x)

			// Compute via Sqr(x)
			bufSqr := make(fermat, 8*n)
			resSqr := bufSqr.Sqr(x)

			// Compare
			if len(resMul) != len(resSqr) {
				t.Fatalf("n=%d: length mismatch: Mul=%d, Sqr=%d", n, len(resMul), len(resSqr))
			}
			for i := range resMul {
				if resMul[i] != resSqr[i] {
					t.Fatalf("n=%d: word %d mismatch: Mul=%x, Sqr=%x", n, i, resMul[i], resSqr[i])
				}
			}
		})
	}
}

// TestFermatSqrZero verifies that squaring a zero fermat number produces zero.
func TestFermatSqrZero(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 5, 10, 30, 50} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			x := make(fermat, n+1) // all zeros
			buf := make(fermat, 8*n)
			res := buf.Sqr(x)
			for i, w := range res {
				if w != 0 {
					t.Fatalf("n=%d: Sqr(0) non-zero at word %d: %x", n, i, w)
				}
			}
		})
	}
}

// TestFermatSqrOne verifies squaring when x = 1 (only first word is 1).
func TestFermatSqrOne(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 5, 10, 30, 50} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			x := make(fermat, n+1)
			x[0] = 1

			bufMul := make(fermat, 8*n)
			resMul := bufMul.Mul(x, x)

			bufSqr := make(fermat, 8*n)
			resSqr := bufSqr.Sqr(x)

			for i := range resMul {
				if resMul[i] != resSqr[i] {
					t.Fatalf("n=%d: word %d mismatch: Mul=%x, Sqr=%x", n, i, resMul[i], resSqr[i])
				}
			}
		})
	}
}

// TestFermatSqrMaxWord verifies squaring when all words are max value.
func TestFermatSqrMaxWord(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 5, 10, 29, 30, 31} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			x := make(fermat, n+1)
			for j := 0; j < n; j++ {
				x[j] = ^big.Word(0)
			}
			x[n] = 1

			bufMul := make(fermat, 8*n)
			resMul := bufMul.Mul(x, x)

			bufSqr := make(fermat, 8*n)
			resSqr := bufSqr.Sqr(x)

			for i := range resMul {
				if resMul[i] != resSqr[i] {
					t.Fatalf("n=%d: word %d mismatch: Mul=%x, Sqr=%x", n, i, resMul[i], resSqr[i])
				}
			}
		})
	}
}

// TestBasicSqrVsBasicMul verifies basicSqr directly against basicMul for small sizes.
func TestBasicSqrVsBasicMul(t *testing.T) {
	t.Parallel()

	for n := 1; n < smallMulThreshold; n++ {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			// Per-subtest source: *rand.Rand is not safe for concurrent
			// use across the parallel subtests (-race).
			rng := rand.New(rand.NewSource(123 + int64(n)))
			x := make(fermat, n)
			for j := 0; j < n; j++ {
				x[j] = big.Word(rng.Uint64())
			}

			zMul := make(fermat, 2*n)
			basicMul(zMul, x, x)

			zSqr := make(fermat, 2*n)
			basicSqr(zSqr, x)

			for i := range zMul {
				if zMul[i] != zSqr[i] {
					t.Fatalf("n=%d: word %d mismatch: basicMul=%x, basicSqr=%x", n, i, zMul[i], zSqr[i])
				}
			}
		})
	}
}

// TestFermatSafeWrappersReturnError verifies that the *Safe wrappers (audit
// R3.8) return a descriptive error on size-mismatched inputs instead of
// panicking. The non-Safe variants are expected to keep panicking and are
// covered by the existing happy-path tests above.
func TestFermatSafeWrappersReturnError(t *testing.T) {
	t.Parallel()

	// Build operands with intentionally mismatched sizes. The exact contents
	// don't matter — the wrappers must reject the call before any arithmetic
	// happens.
	x := make(fermat, 10) // n+1 = 10  → n = 9
	y := make(fermat, 12) // n+1 = 12  → n = 11
	z := make(fermat, 10)

	t.Run("MulSafe rejects mismatched len(x), len(y)", func(t *testing.T) {
		t.Parallel()
		buf := make(fermat, 8*9)
		_, err := buf.MulSafe(x, y)
		if err == nil {
			t.Fatalf("MulSafe: expected error on len(x)=%d != len(y)=%d, got nil", len(x), len(y))
		}
		if !strings.Contains(err.Error(), "operand size mismatch") {
			t.Fatalf("MulSafe: error message missing context: %v", err)
		}
	})

	t.Run("ShiftSafe rejects mismatched len(z), len(x)", func(t *testing.T) {
		t.Parallel()
		err := y.ShiftSafe(x, 3) // len(y)=12 != len(x)=10
		if err == nil {
			t.Fatalf("ShiftSafe: expected error on len(z)=%d != len(x)=%d, got nil", len(y), len(x))
		}
		if !strings.Contains(err.Error(), "operand size mismatch") {
			t.Fatalf("ShiftSafe: error message missing context: %v", err)
		}
	})

	t.Run("AddSafe rejects mismatched len(z), len(x)", func(t *testing.T) {
		t.Parallel()
		_, err := z.AddSafe(y, z) // len(z)=10 != len(y)=12
		if err == nil {
			t.Fatalf("AddSafe: expected error on len(z)=%d != len(x)=%d, got nil", len(z), len(y))
		}
		if !strings.Contains(err.Error(), "operand size mismatch") {
			t.Fatalf("AddSafe: error message missing context: %v", err)
		}
	})

	t.Run("SubSafe rejects mismatched len(z), len(y)", func(t *testing.T) {
		t.Parallel()
		_, err := z.SubSafe(z, y) // len(z)=10 != len(y)=12
		if err == nil {
			t.Fatalf("SubSafe: expected error on len(z)=%d != len(y)=%d, got nil", len(z), len(y))
		}
		if !strings.Contains(err.Error(), "operand size mismatch") {
			t.Fatalf("SubSafe: error message missing context: %v", err)
		}
	})

	t.Run("SqrSafe rejects empty operand", func(t *testing.T) {
		t.Parallel()
		buf := make(fermat, 8)
		_, err := buf.SqrSafe(fermat{})
		if err == nil {
			t.Fatalf("SqrSafe: expected error on empty operand, got nil")
		}
	})

	// Happy path: matched sizes still work and produce the same result as the
	// underlying panic-based methods. This guards against the wrappers ever
	// silently diverging from Mul/Sqr/Add/Sub/Shift.
	t.Run("MulSafe matches Mul on valid input", func(t *testing.T) {
		t.Parallel()
		n := 5
		a := make(fermat, n+1)
		b := make(fermat, n+1)
		for i := 0; i < n; i++ {
			a[i] = big.Word(i*7 + 3)
			b[i] = big.Word(i*11 + 5)
		}
		bufRef := make(fermat, 8*n)
		bufSafe := make(fermat, 8*n)
		ref := bufRef.Mul(a, b)
		got, err := bufSafe.MulSafe(a, b)
		if err != nil {
			t.Fatalf("MulSafe on valid input returned error: %v", err)
		}
		if len(ref) != len(got) {
			t.Fatalf("MulSafe length mismatch with Mul: %d vs %d", len(got), len(ref))
		}
		for i := range ref {
			if ref[i] != got[i] {
				t.Fatalf("MulSafe diverges from Mul at word %d: got %x, want %x", i, got[i], ref[i])
			}
		}
	})
}

// BenchmarkFermatSqrVsMul benchmarks fermat.Sqr vs fermat.Mul at sizes
// below and above smallMulThreshold.
func BenchmarkFermatSqrVsMul(b *testing.B) {
	rng := rand.New(rand.NewSource(42))

	for _, n := range []int{10, 29, 30, 50} {
		x := make(fermat, n+1)
		for j := 0; j < n; j++ {
			x[j] = big.Word(rng.Uint64())
		}

		b.Run(fmt.Sprintf("n=%d/Mul", n), func(b *testing.B) {
			buf := make(fermat, 8*n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf.Mul(x, x)
			}
		})

		b.Run(fmt.Sprintf("n=%d/Sqr", n), func(b *testing.B) {
			buf := make(fermat, 8*n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf.Sqr(x)
			}
		})
	}
}
