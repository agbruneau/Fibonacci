package fibonacci

import (
	"context"
	"testing"
)

// TestMatrixExponentiation_FFTPathIsolation closes an audit coverage gap:
// the golden test compares all calculators at DEFAULT thresholds, where the
// matrix path may never route a multiplication through the FFT tier. If the
// matrix x FFT coupling broke, every existing test would still pass. This
// test forces the routing decision both ways on the same input and requires
// identical results, so a regression in the matrix FFT path (or in the
// Strassen + FFT combination) fails loudly.
//
// N is chosen so the top matrix elements (~F(N), about 208k bits for
// N=300k) comfortably exceed bigfft's internal word threshold (~115k bits),
// guaranteeing real FFT transforms run — not just the fibonacci-level
// routing into bigfft's fallback.
func TestMatrixExponentiation_FFTPathIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FFT-path isolation (N=300k) in -short mode")
	}
	t.Parallel()

	const n = 300_000
	calc := &MatrixExponentiationCalculator{}
	ctx := context.Background()

	run := func(name string, opts Options) string {
		t.Helper()
		res, err := calc.CalculateCore(ctx, noopReporter, n, opts)
		if err != nil {
			t.Fatalf("%s: CalculateCore(F(%d)) failed: %v", name, n, err)
		}
		return res.Text(16)
	}

	// Reference: FFT and Strassen disabled (thresholds above any operand).
	want := run("fft-off", Options{
		FFTThreshold:      1 << 30,
		ParallelThreshold: 4096,
		StrassenThreshold: 1 << 30,
	})

	// FFT forced: every multiplication above 64 bits routes through the
	// bigfft tier of smartMultiply/smartSquare.
	if got := run("fft-forced", Options{
		FFTThreshold:      64,
		ParallelThreshold: 4096,
		StrassenThreshold: 1 << 30,
	}); got != want {
		t.Error("matrix exponentiation result differs when FFT path is forced")
	}

	// FFT + Strassen forced together: exercises the executeTasks /
	// executeMixedTasks batch dispatch with FFT-tier multiplications.
	if got := run("fft+strassen-forced", Options{
		FFTThreshold:      64,
		ParallelThreshold: 4096,
		StrassenThreshold: 128,
	}); got != want {
		t.Error("matrix exponentiation result differs when FFT and Strassen paths are forced")
	}
}
