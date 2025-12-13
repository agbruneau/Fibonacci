package bigfft

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Multiplication Benchmarks (varying sizes)
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkMul100K benchmarks FFT multiplication of ~100K bit numbers.
func BenchmarkMul100K(b *testing.B) {
	benchmarkMul(b, 12500) // 12500 bytes = 100K bits
}

// BenchmarkMul500K benchmarks FFT multiplication of ~500K bit numbers.
func BenchmarkMul500K(b *testing.B) {
	benchmarkMul(b, 62500) // 62500 bytes = 500K bits
}

// BenchmarkMul1M benchmarks FFT multiplication of ~1M bit numbers.
func BenchmarkMul1M(b *testing.B) {
	benchmarkMul(b, 125000) // 125000 bytes = 1M bits
}

// BenchmarkMul5M benchmarks FFT multiplication of ~5M bit numbers.
func BenchmarkMul5M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 5M benchmark in short mode")
	}
	benchmarkMul(b, 625000) // 625000 bytes = 5M bits
}

// BenchmarkMul10M benchmarks FFT multiplication of ~10M bit numbers.
func BenchmarkMul10M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 10M benchmark in short mode")
	}
	benchmarkMul(b, 1250000) // 1.25M bytes = 10M bits
}

func benchmarkMul(b *testing.B, byteSize int) {
	xBytes := make([]byte, byteSize)
	yBytes := make([]byte, byteSize)
	rand.Read(xBytes)
	rand.Read(yBytes)
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	b.ReportAllocs()
	b.SetBytes(int64(byteSize * 2))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Mul(x, y)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MulTo Benchmarks (with buffer reuse)
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkMulTo100K benchmarks MulTo with buffer reuse.
func BenchmarkMulTo100K(b *testing.B) {
	benchmarkMulTo(b, 12500)
}

// BenchmarkMulTo500K benchmarks MulTo with buffer reuse.
func BenchmarkMulTo500K(b *testing.B) {
	benchmarkMulTo(b, 62500)
}

// BenchmarkMulTo1M benchmarks MulTo with buffer reuse.
func BenchmarkMulTo1M(b *testing.B) {
	benchmarkMulTo(b, 125000)
}

func benchmarkMulTo(b *testing.B, byteSize int) {
	xBytes := make([]byte, byteSize)
	yBytes := make([]byte, byteSize)
	rand.Read(xBytes)
	rand.Read(yBytes)
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)
	z := new(big.Int)

	b.ReportAllocs()
	b.SetBytes(int64(byteSize * 2))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = MulTo(z, x, y)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Squaring Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkSqr100K benchmarks FFT squaring of ~100K bit numbers.
func BenchmarkSqr100K(b *testing.B) {
	benchmarkSqr(b, 12500)
}

// BenchmarkSqr500K benchmarks FFT squaring of ~500K bit numbers.
func BenchmarkSqr500K(b *testing.B) {
	benchmarkSqr(b, 62500)
}

// BenchmarkSqr1M benchmarks FFT squaring of ~1M bit numbers.
func BenchmarkSqr1M(b *testing.B) {
	benchmarkSqr(b, 125000)
}

// BenchmarkSqr5M benchmarks FFT squaring of ~5M bit numbers.
func BenchmarkSqr5M(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping 5M benchmark in short mode")
	}
	benchmarkSqr(b, 625000)
}

func benchmarkSqr(b *testing.B, byteSize int) {
	xBytes := make([]byte, byteSize)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)

	b.ReportAllocs()
	b.SetBytes(int64(byteSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Sqr(x)
	}
}

// BenchmarkSqrTo100K benchmarks SqrTo with buffer reuse.
func BenchmarkSqrTo100K(b *testing.B) {
	benchmarkSqrTo(b, 12500)
}

// BenchmarkSqrTo500K benchmarks SqrTo with buffer reuse.
func BenchmarkSqrTo500K(b *testing.B) {
	benchmarkSqrTo(b, 62500)
}

// BenchmarkSqrTo1M benchmarks SqrTo with buffer reuse.
func BenchmarkSqrTo1M(b *testing.B) {
	benchmarkSqrTo(b, 125000)
}

func benchmarkSqrTo(b *testing.B, byteSize int) {
	xBytes := make([]byte, byteSize)
	rand.Read(xBytes)
	x := new(big.Int).SetBytes(xBytes)
	z := new(big.Int)

	b.ReportAllocs()
	b.SetBytes(int64(byteSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = SqrTo(z, x)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Low-level Arithmetic Benchmarks (comparing linkname vs Auto dispatch)
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkArithAddVVLinkname benchmarks vector-vector addition using go:linkname.
func BenchmarkArithAddVVLinkname(b *testing.B) {
	benchmarkAddVVSize(b, 1000)
}

// BenchmarkArithAddVVLinknameLarge benchmarks vector-vector addition with larger vectors.
func BenchmarkArithAddVVLinknameLarge(b *testing.B) {
	benchmarkAddVVSize(b, 10000)
}

// BenchmarkArithAddVVAutoDispatch benchmarks auto-dispatched vector-vector addition.
func BenchmarkArithAddVVAutoDispatch(b *testing.B) {
	benchmarkAddVVAutoSize(b, 1000)
}

// BenchmarkArithAddVVAutoDispatchLarge benchmarks auto-dispatched vector-vector addition.
func BenchmarkArithAddVVAutoDispatchLarge(b *testing.B) {
	benchmarkAddVVAutoSize(b, 10000)
}

func benchmarkAddVVSize(b *testing.B, size int) {
	x := make([]big.Word, size)
	y := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i + 1)
		y[i] = big.Word(i * 2)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size * 8)) // 8 bytes per Word on 64-bit
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		addVV(z, x, y)
	}
}

func benchmarkAddVVAutoSize(b *testing.B, size int) {
	x := make([]big.Word, size)
	y := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i + 1)
		y[i] = big.Word(i * 2)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size * 8))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		AddVVAuto(z, x, y)
	}
}

// BenchmarkArithSubVVLinkname benchmarks vector-vector subtraction.
func BenchmarkArithSubVVLinkname(b *testing.B) {
	size := 1000
	x := make([]big.Word, size)
	y := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i*2 + 100)
		y[i] = big.Word(i)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size * 8))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		subVV(z, x, y)
	}
}

// BenchmarkArithSubVVAutoDispatch benchmarks auto-dispatched vector-vector subtraction.
func BenchmarkArithSubVVAutoDispatch(b *testing.B) {
	size := 1000
	x := make([]big.Word, size)
	y := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i*2 + 100)
		y[i] = big.Word(i)
	}

	b.ReportAllocs()
	b.SetBytes(int64(size * 8))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SubVVAuto(z, x, y)
	}
}

// BenchmarkArithAddMulVVWLinkname benchmarks multiply-accumulate.
func BenchmarkArithAddMulVVWLinkname(b *testing.B) {
	size := 1000
	x := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i + 1)
	}
	y := big.Word(12345)

	b.ReportAllocs()
	b.SetBytes(int64(size * 8))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Reset z to avoid overflow accumulation
		for j := range z {
			z[j] = 0
		}
		addMulVVW(z, x, y)
	}
}

// BenchmarkArithAddMulVVWAutoDispatch benchmarks auto-dispatched multiply-accumulate.
func BenchmarkArithAddMulVVWAutoDispatch(b *testing.B) {
	size := 1000
	x := make([]big.Word, size)
	z := make([]big.Word, size)
	for i := range x {
		x[i] = big.Word(i + 1)
	}
	y := big.Word(12345)

	b.ReportAllocs()
	b.SetBytes(int64(size * 8))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for j := range z {
			z[j] = 0
		}
		AddMulVVWAuto(z, x, y)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Bump Allocator Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkBumpAlloc benchmarks bump allocator performance.
func BenchmarkBumpAlloc(b *testing.B) {
	capacity := 100000
	allocSize := 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ba := AcquireBumpAllocator(capacity)
		for j := 0; j < 50; j++ {
			_ = ba.Alloc(allocSize)
		}
		ReleaseBumpAllocator(ba)
	}
}

// BenchmarkBumpAllocUnsafe benchmarks unsafe bump allocation.
func BenchmarkBumpAllocUnsafe(b *testing.B) {
	capacity := 100000
	allocSize := 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ba := AcquireBumpAllocator(capacity)
		for j := 0; j < 50; j++ {
			_ = ba.AllocUnsafe(allocSize)
		}
		ReleaseBumpAllocator(ba)
	}
}

// BenchmarkPoolAcquireRelease benchmarks sync.Pool acquire/release overhead.
func BenchmarkPoolAcquireRelease(b *testing.B) {
	size := 1000

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		f := acquireFermat(size)
		releaseFermat(f)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Transform Benchmarks
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkTransform benchmarks polynomial FFT transform.
func BenchmarkTransform(b *testing.B) {
	k := uint(8) // FFT size 256
	m := 10
	n := valueSize(k, m, 2)

	p := poly{k: k, m: m}
	p.a = make([]nat, 1<<k)
	for i := range p.a {
		p.a[i] = make(nat, m)
		for j := range p.a[i] {
			p.a[i][j] = big.Word(i*m + j + 1)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = p.Transform(n)
	}
}

// BenchmarkTransformWithBump benchmarks polynomial FFT transform with bump allocator.
func BenchmarkTransformWithBump(b *testing.B) {
	k := uint(8)
	m := 10
	n := valueSize(k, m, 2)

	p := poly{k: k, m: m}
	p.a = make([]nat, 1<<k)
	for i := range p.a {
		p.a[i] = make(nat, m)
		for j := range p.a[i] {
			p.a[i][j] = big.Word(i*m + j + 1)
		}
	}

	wordLen := len(p.a) * m
	capacity := EstimateBumpCapacity(wordLen)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ba := AcquireBumpAllocator(capacity)
		_, _ = p.TransformWithBump(n, ba)
		ReleaseBumpAllocator(ba)
	}
}
