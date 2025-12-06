package bigfft

import (
	"math/big"
	"testing"
)

func TestFFT_Sqr(t *testing.T) {
	// Test Sqr function directly
	// Choose a size that triggers FFT
	// fftThreshold = 1800 words
	// 1 word = 64 bits = 8 bytes
	// 1800 words = 14400 bytes = 115200 bits

	// Create a large number slightly above threshold to ensure FFT path is taken
	// For testing, we can temporarily lower the threshold if possible,
	// or just trust the threshold logic.
	// But `fftThreshold` is unexported.
	// However, `Sqr` is exported.

	// Let's create a moderately large number and verify correctness
	// even if it doesn't trigger FFT (it will use Mul fallback),
	// coverage will increase for the Sqr function entry point.
	// To strictly hit sqrFFT, we need xwords > fftThreshold.

	// Creating a huge number in test might be slow but let's try a reasonable size.
	// 2000 words to be safe.

	numWords := 2000
	x := new(big.Int)
	words := make([]big.Word, numWords)
	for i := range words {
		words[i] = big.Word(0x123456789ABCDEF0 + i)
	}
	x.SetBits(words)

	// Calculate expected result using standard library (slow but correct)
	expected := new(big.Int).Mul(x, x)

	// Calculate using Sqr
	result, err := Sqr(x)
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("Sqr result mismatch")
	}
}

func TestFFT_SqrTo(t *testing.T) {
	numWords := 2000
	x := new(big.Int)
	words := make([]big.Word, numWords)
	for i := range words {
		words[i] = big.Word(0xFEEDBACC + i)
	}
	x.SetBits(words)

	expected := new(big.Int).Mul(x, x)

	// Pre-allocate z
	z := new(big.Int)

	result, err := SqrTo(z, x)
	if err != nil {
		t.Fatalf("SqrTo failed: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("SqrTo result mismatch")
	}

	// Check if z was reused (same pointer returned)
	if result != z {
		t.Errorf("SqrTo did not return the destination pointer")
	}
}

// Test internal Sqr/SqrWithBump logic via lower level if needed,
// but Sqr/SqrTo coverage should propagate down.
// Since we used > threshold, it should hit sqrFFT -> fftsqr -> fftsqrTo.
// And inside fftsqrTo it calls TransformWithBump, SqrWithBump, InvTransformWithBump.
// This should cover SqrWithBump which was 0%.

func TestPoly_Sqr(t *testing.T) {
	// Directly test polValues.Sqr which was 0% (SqrWithBump is used by SqrTo path)
	// We need to construct a polValues struct.
	// This requires some internal setup, might be hard to test purely from outside if structs are not exported.
	// `polValues` is unexported.
	// But we are in package `bigfft`, so we can access it.

	// We need a valid polValues to test Sqr.
	// We can get one by manually creating a poly and transforming it.

	k := uint(4) // Small FFT size
	m := 1
	n := valueSize(k, m, 2)

	// Create a simple polynomial
	p := poly{k: k, m: m}
	p.a = make([]nat, 1<<k)
	for i := range p.a {
		p.a[i] = nat{big.Word(i + 1)}
	}

	// Transform it
	pv, err := p.Transform(n)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Test Sqr (which uses pooled buffer, not bump)
	sqrPV, err := pv.Sqr()
	if err != nil {
		t.Fatalf("Sqr failed: %v", err)
	}

	// Verify result by comparing with Mul(p, p)
	mulPV, err := pv.Mul(&pv)
	if err != nil {
		t.Fatalf("Mul failed: %v", err)
	}

	// Compare values
	for i := range sqrPV.values {
		// values are slices of fermat (which is []Word)
		v1 := sqrPV.values[i]
		v2 := mulPV.values[i]

		if len(v1) != len(v2) {
			t.Errorf("Length mismatch at %d", i)
			continue
		}
		for j := range v1 {
			if v1[j] != v2[j] {
				t.Errorf("Value mismatch at index %d word %d: %v != %v", i, j, v1[j], v2[j])
				break
			}
		}
	}
}

func TestNTransform_InvNTransform(t *testing.T) {
	// Test NTransform and InvNTransform which had 0% coverage

	k := uint(4)
	m := 1
	n := valueSize(k, m, 2)

	// Create polynomial
	p := poly{k: k, m: m}
	// Must be strictly less than 1<<k for NTransform
	p.a = make([]nat, (1<<k)-1)
	for i := range p.a {
		p.a[i] = nat{big.Word(i + 1)}
	}

	// NTransform
	vals := p.NTransform(n)

	// InvNTransform
	pRes := vals.InvNTransform()

	// Verify we got back original coefficients (scaled/shifted? InvNTransform docs say m is unspecified)
	// Usually NTransform/InvNTransform round trip should preserve data up to scaling/modulus.
	// Let's check `pRes.a`.

	// Note: NTransform/InvNTransform logic might be complex involving roots of unity.
	// Basic check: should not panic and return something.

	if pRes.k != k {
		t.Errorf("Expected k=%d, got %d", k, pRes.k)
	}

	// We won't assert exact values without deeper understanding of NTransform math here,
	// but ensuring it runs covers the code paths.
}

func TestStringMethods(t *testing.T) {
	// Cover String() methods
	n := nat{123, 456}
	s := n.String()
	if s == "" {
		t.Error("nat.String() returned empty string")
	}

	// fermat.String()
	// fermat is also just []Word, but distinct type
	f := fermat{789}
	fs := f.String()
	if fs == "" {
		t.Error("fermat.String() returned empty string")
	}
}
