package bigfft

import (
	"math/big"
	"testing"
)

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
		// 4096 bytes = 512 words, still well under defaultFFTThresholdWords
		// (1800): this seed exercises the large-operand math/big path, not
		// the FFT one. Only fuzzer-grown inputs above ~14 400 bytes on both
		// operands reach mulFFT.
		{makeRepeated(0xff, 4096), makeRepeated(0xff, 4096)},
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
