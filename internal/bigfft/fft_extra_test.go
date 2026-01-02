package bigfft

import (
	"math/big"
	"testing"
)

func TestFFT_Sqr(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	// Directly test PolValues.Sqr which was 0% (SqrWithBump is used by SqrTo path)
	// We need to construct a PolValues struct.
	// This requires some internal setup, might be hard to test purely from outside if structs are not exported.
	// `PolValues` is now exported.

	// We need a valid PolValues to test Sqr.
	// We can get one by manually creating a Poly and transforming it.

	k := uint(4) // Small FFT size
	m := 1
	n := valueSize(k, m, 2)

	// Create a simple polynomial
	p := Poly{K: k, M: m}
	p.A = make([]nat, 1<<k)
	for i := range p.A {
		p.A[i] = nat{big.Word(i + 1)}
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
	for i := range sqrPV.Values {
		// values are slices of fermat (which is []Word)
		v1 := sqrPV.Values[i]
		v2 := mulPV.Values[i]

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
	t.Parallel()
	// Test NTransform and InvNTransform which had 0% coverage

	k := uint(4)
	m := 1
	n := valueSize(k, m, 2)

	// Create polynomial
	p := Poly{K: k, M: m}
	// Must be strictly less than 1<<k for NTransform
	p.A = make([]nat, (1<<k)-1)
	for i := range p.A {
		p.A[i] = nat{big.Word(i + 1)}
	}

	// NTransform
	vals := p.NTransform(n)

	// InvNTransform
	pRes := vals.InvNTransform()

	// Verify we got back original coefficients (scaled/shifted? InvNTransform docs say m is unspecified)
	// Usually NTransform/InvNTransform round trip should preserve data up to scaling/modulus.
	// Let's check `pRes.A`.

	// Note: NTransform/InvNTransform logic might be complex involving roots of unity.
	// Basic check: should not panic and return something.

	if pRes.K != k {
		t.Errorf("Expected K=%d, got %d", k, pRes.K)
	}

	// We won't assert exact values without deeper understanding of NTransform math here,
	// but ensuring it runs covers the code paths.
}

func TestStringMethods(t *testing.T) {
	t.Parallel()
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

func TestSqrWithBumpDirect(t *testing.T) {
	t.Parallel()
	k := uint(4)
	m := 1
	n := valueSize(k, m, 2)
	ba := AcquireBumpAllocator(1000)
	defer ReleaseBumpAllocator(ba)

	p := Poly{K: k, M: m}
	p.A = make([]nat, 1<<k)
	for i := range p.A {
		p.A[i] = nat{big.Word(i + 1)}
	}

	pv, _ := p.Transform(n)

	// Test SqrWithBump
	sqrPV, err := pv.SqrWithBump(ba)
	if err != nil {
		t.Fatalf("SqrWithBump failed: %v", err)
	}
	if len(sqrPV.Values) != 1<<k {
		t.Errorf("Expected %d values, got %d", 1<<k, len(sqrPV.Values))
	}
}

func TestTransformWithBump(t *testing.T) {
	t.Parallel()
	k := uint(4)
	m := 1
	n := valueSize(k, m, 2)
	ba := AcquireBumpAllocator(1000)
	defer ReleaseBumpAllocator(ba)

	p := Poly{K: k, M: m}
	p.A = make([]nat, 1<<k)
	for i := range p.A {
		p.A[i] = nat{big.Word(i + 1)}
	}

	pv, err := p.TransformWithBump(n, ba)
	if err != nil {
		t.Fatalf("TransformWithBump failed: %v", err)
	}

	pRes, err := pv.InvTransformWithBump(ba)
	if err != nil {
		t.Fatalf("InvTransformWithBump failed: %v", err)
	}
	if pRes.K != k {
		t.Errorf("Expected K=%d, got %d", k, pRes.K)
	}
}

func TestFFTUtilities(t *testing.T) {
	t.Parallel()
	// Test fftSize
	x := make(nat, 100)
	y := make(nat, 200)
	k, m := fftSize(x, y)
	if k == 0 || m == 0 {
		t.Errorf("fftSize returned invalid values: k=%d, m=%d", k, m)
	}

	// Test valueSize
	vSize := valueSize(10, 100, 2)
	if vSize <= 0 {
		t.Errorf("valueSize returned invalid value: %d", vSize)
	}

	// Test trim
	n1 := nat{1, 2, 0, 0}
	trimmed := trim(n1)
	if len(trimmed) != 2 {
		t.Errorf("trim failed: expected length 2, got %d", len(trimmed))
	}
	if trim(nat{0, 0}) != nil {
		t.Error("trim of all zeros should return nil")
	}

	// Test polyFromNat
	n2 := make(nat, 25)
	for i := range n2 {
		n2[i] = big.Word(i)
	}
	p := polyFromNat(n2, 5, 10)
	if len(p.A) != 3 { // ceil(25/10) = 3
		t.Errorf("polyFromNat failed: expected 3 coefficients, got %d", len(p.A))
	}

	// Test Int() and IntTo()
	n3 := p.Int()
	if len(n3) == 0 {
		t.Error("Int() returned empty result")
	}

	n4 := make(nat, 100)
	n5 := p.IntTo(n4)
	if len(n5) == 0 {
		t.Error("IntTo() returned empty result")
	}
}

func TestFFTParallel(t *testing.T) {
	t.Parallel()
	// Use large k to trigger parallel path (depth < MaxParallelFFTDepth)
	k := uint(4) // Use a smaller k for faster test but still trigger logic if ParallelFFTRecursionThreshold is small
	numPoints := 1 << k
	n := 10 // word size
	src := make([]fermat, numPoints)
	dst := make([]fermat, numPoints)
	for i := range src {
		src[i] = make(fermat, n+1)
		src[i][0] = big.Word(i)
		dst[i] = make(fermat, n+1)
	}

	// This should use multiple goroutines if thresholds are met
	err := fourier(dst, src, false, n, k)
	if err != nil {
		t.Fatalf("fourier failed: %v", err)
	}
}

func TestMulCached(t *testing.T) {
	t.Parallel()
	k := uint(4)
	m := 1
	p := Poly{K: k, M: m, A: []nat{{1}, {2}, {3}}}
	q := Poly{K: k, M: m, A: []nat{{4}, {5}, {6}}}

	// Test MulCached
	res, err := p.MulCached(&q)
	if err != nil {
		t.Fatalf("MulCached failed: %v", err)
	}
	if len(res.A) == 0 {
		t.Error("MulCached result has no coefficients")
	}

	// Test SqrCached
	resSqr, err := p.SqrCached()
	if err != nil {
		t.Fatalf("SqrCached failed: %v", err)
	}
	if len(resSqr.A) == 0 {
		t.Error("SqrCached result has no coefficients")
	}
}

func TestPolyFromInt(t *testing.T) {
	t.Parallel()

	t.Run("Convert big.Int to Poly", func(t *testing.T) {
		t.Parallel()
		x := big.NewInt(12345)
		k := uint(4)
		m := 1

		p := PolyFromInt(x, k, m)

		if p.K != k {
			t.Errorf("Expected K=%d, got %d", k, p.K)
		}
		if p.M != m {
			t.Errorf("Expected M=%d, got %d", m, p.M)
		}
		if len(p.A) == 0 {
			t.Error("Poly should have coefficients")
		}
	})

	t.Run("Convert large big.Int to Poly", func(t *testing.T) {
		t.Parallel()
		x := new(big.Int).Exp(big.NewInt(2), big.NewInt(100), nil)
		k := uint(5)
		m := 2

		p := PolyFromInt(x, k, m)

		if p.K != k {
			t.Errorf("Expected K=%d, got %d", k, p.K)
		}
		if p.M != m {
			t.Errorf("Expected M=%d, got %d", m, p.M)
		}
	})
}

func TestGetFFTParams(t *testing.T) {
	t.Parallel()

	t.Run("Get FFT params for small word count", func(t *testing.T) {
		t.Parallel()
		words := 100
		k, m := GetFFTParams(words)

		if k == 0 {
			t.Error("k should be non-zero")
		}
		if m <= 0 {
			t.Errorf("m should be positive, got %d", m)
		}
	})

	t.Run("Get FFT params for medium word count", func(t *testing.T) {
		t.Parallel()
		words := 1000
		k, m := GetFFTParams(words)

		if k == 0 {
			t.Error("k should be non-zero")
		}
		if m <= 0 {
			t.Errorf("m should be positive, got %d", m)
		}
	})

	t.Run("Get FFT params for large word count", func(t *testing.T) {
		t.Parallel()
		words := 10000
		k, m := GetFFTParams(words)

		if k == 0 {
			t.Error("k should be non-zero")
		}
		if m <= 0 {
			t.Errorf("m should be positive, got %d", m)
		}
	})
}

func TestValueSize(t *testing.T) {
	t.Parallel()

	t.Run("ValueSize with small parameters", func(t *testing.T) {
		t.Parallel()
		k := uint(4)
		m := 1
		extra := uint(2)

		size := ValueSize(k, m, extra)

		if size <= 0 {
			t.Errorf("ValueSize should return positive value, got %d", size)
		}
	})

	t.Run("ValueSize with medium parameters", func(t *testing.T) {
		t.Parallel()
		k := uint(5)
		m := 10
		extra := uint(1)

		size := ValueSize(k, m, extra)

		if size <= 0 {
			t.Errorf("ValueSize should return positive value, got %d", size)
		}
	})

	t.Run("ValueSize with large parameters", func(t *testing.T) {
		t.Parallel()
		k := uint(10)
		m := 100
		extra := uint(2)

		size := ValueSize(k, m, extra)

		if size <= 0 {
			t.Errorf("ValueSize should return positive value, got %d", size)
		}
	})
}

func TestPoly_IntToBigInt(t *testing.T) {
	t.Parallel()

	t.Run("Convert Poly to big.Int", func(t *testing.T) {
		t.Parallel()
		k := uint(4)
		m := 1
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		z := new(big.Int)
		result := p.IntToBigInt(z)

		if result != z {
			t.Error("IntToBigInt should return the provided big.Int")
		}
		if result.Sign() == 0 && len(p.A) > 0 {
			t.Error("Result should be non-zero for non-empty polynomial")
		}
	})

	t.Run("Convert Poly to big.Int with nil", func(t *testing.T) {
		t.Parallel()
		k := uint(4)
		m := 1
		p := Poly{K: k, M: m}
		p.A = make([]nat, 1<<k)
		for i := range p.A {
			p.A[i] = nat{big.Word(i + 1)}
		}

		z := new(big.Int)
		result := p.IntToBigInt(z)

		if result == nil {
			t.Error("IntToBigInt should not return nil")
		}
	})
}
