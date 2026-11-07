package bigfft

import (
	"math/big"
	"math/bits"
	"testing"
)

// The FFT path is the reason these fuzzers exist, and until 2026-09-07 no seed
// reached it (audit TST-04). Mul takes it only when BOTH operands exceed
// getFFTThreshold() words; the largest seed was 4096 bytes = 512 words against
// a 1800-word threshold, so every replay of the seed corpus — which is what
// `go test` does, mutation fuzzing having never been scheduled — compared
// math/big against math/big.
//
// fftSeed sizes its operands in WORDS, relative to the real threshold, so the
// seeds stay on the FFT side of the branch on a 32-bit target too (where a word
// is 4 bytes) and follow the constant if it is ever retuned.
func fftSeed(extraWords int, fill byte) []byte {
	const bytesPerWord = bits.UintSize / 8
	n := (defaultFFTThresholdWords + extraWords) * bytesPerWord
	b := make([]byte, n)
	for i := range b {
		// Vary the bytes: a constant fill produces uniform limbs, which is the
		// easiest possible carry pattern for both implementations.
		b[i] = fill ^ byte(i*31)
	}
	// Guarantee a set top bit, so SetBytes yields exactly the intended word
	// count instead of one fewer when the leading byte happens to xor to zero.
	b[0] |= 0x80
	return b
}

// FuzzMul exercises the bigfft.Mul entry point directly with adversarial
// operand pairs. The reference is math/big.Int.Mul which is exhaustively
// tested upstream; any divergence in the FFT path surfaces here.
// Audit-PRD E8-R2 / Sprint S1-T9.
func FuzzMul(f *testing.F) {
	seeds := [][2][]byte{
		{{0x01}, {0x02}},
		{{0xff}, {0xff}},
		{{0x80, 0x00, 0x00, 0x00}, {0x80, 0x00, 0x00, 0x00}},
		{makeRepeated(0xa5, 256), makeRepeated(0x5a, 256)},
		// 512 words: the largest operand pair that still takes the math/big
		// path, which is worth keeping — it pins the branch just below the
		// crossover.
		{makeRepeated(0xff, 4096), makeRepeated(0xff, 4096)},
		// Both operands one word past the threshold: the FFT branch at its
		// boundary, the case most likely to expose an off-by-one in sizing.
		{fftSeed(1, 0xa5), fftSeed(1, 0x5a)},
		// Comfortably inside the FFT regime, with different fills so the two
		// operands are not near-identical.
		{fftSeed(1200, 0x3c), fftSeed(1200, 0xc3)},
		// Asymmetric: one operand over the threshold, one under. Mul requires
		// BOTH to be over, so this must still agree with math/big through the
		// non-FFT branch.
		{fftSeed(64, 0x11), makeRepeated(0x22, 256)},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}

	f.Fuzz(func(t *testing.T, a, b []byte) {
		// Bound input size to keep the harness responsive while still
		// hitting the FFT regime (default fftThreshold = 1800 words on
		// 64-bit = ~14 400 bytes).
		const maxBytes = 32_000
		if len(a) > maxBytes || len(b) > maxBytes {
			return
		}
		if len(a) == 0 || len(b) == 0 {
			return
		}

		x := new(big.Int).SetBytes(a)
		y := new(big.Int).SetBytes(b)

		got, err := Mul(x, y)
		if err != nil {
			// An error is acceptable for adversarial inputs only if it
			// is also reproducible by math/big — which never errors on
			// well-formed Ints. So an error here is a regression.
			t.Fatalf("bigfft.Mul returned error for valid inputs: %v", err)
		}

		want := new(big.Int).Mul(x, y)
		if got.Cmp(want) != 0 {
			t.Fatalf("bigfft.Mul disagrees with math/big.Mul:\n  got len=%d\n  want len=%d", len(got.Bits()), len(want.Bits()))
		}
	})
}

// FuzzSqr exercises bigfft.Sqr with adversarial squarings. Same reference
// as FuzzMul.
func FuzzSqr(f *testing.F) {
	seeds := [][]byte{
		{0x01},
		{0xff},
		{0x80, 0x00, 0x00, 0x00},
		makeRepeated(0xa5, 256),
		makeRepeated(0xff, 4096),
		// Past the threshold, so the seed replay actually reaches sqrFFT
		// (audit TST-04): boundary first, then well inside.
		fftSeed(1, 0xa5),
		fftSeed(1200, 0x3c),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, a []byte) {
		const maxBytes = 32_000
		if len(a) == 0 || len(a) > maxBytes {
			return
		}
		x := new(big.Int).SetBytes(a)
		got, err := Sqr(x)
		if err != nil {
			t.Fatalf("bigfft.Sqr returned error for valid input: %v", err)
		}
		want := new(big.Int).Mul(x, x)
		if got.Cmp(want) != 0 {
			t.Fatalf("bigfft.Sqr disagrees with math/big: got len=%d, want len=%d", len(got.Bits()), len(want.Bits()))
		}
	})
}

// makeRepeated returns a byte slice of length n filled with v. Used to seed
// the fuzzers with high-entropy inputs that push past the FFT activation
// threshold.
func makeRepeated(v byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}
