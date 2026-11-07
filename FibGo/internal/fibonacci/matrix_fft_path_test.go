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
// N must be chosen against bigfft's INPUT operand sizes, not the final
// result: bigfft.MulTo/SqrTo apply their own 1800-word (~115k-bit) gate on
// the operands, and the largest F(N)-sized value is only the OUTPUT of the
// final multiply. Simulating ExecuteMatrixLoop's schedule: at N=600k the
// largest squaring operands are the elements of Q^262144 (~2844 words),
// which pass the gate, yielding 3 FFT-eligible squarings and 1 FFT-eligible
// multiply per run. (At N=300k the max input was 1422 words — every
// operation silently fell back to math/big and the test asserted nothing
// about the transforms.)
func TestMatrixExponentiation_FFTPathIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FFT-path isolation (N=600k) in -short mode")
	}
	t.Parallel()

	const n = 600_000
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
