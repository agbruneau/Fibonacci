// See doc.go for the package-level overview, in particular the
// "Independent oracle" rationale for the fibBig/calculateSmall duplication.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// GoldenData represents a single test case in the golden file
type GoldenData struct {
	N      uint64 `json:"n"`
	Result string `json:"result"`
}

// main writes precomputed Fibonacci values for selected indices to the output
// directory.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// defaultOutDir is the module-relative output directory used when -out is not
// overridden. It only resolves correctly from the repository root.
const defaultOutDir = "internal/fibonacci/testdata"

// run holds the program body so that deferred cleanup (notably file.Close,
// whose error is now surfaced) executes on every error path, which os.Exit in
// main would otherwise skip. The named return lets the deferred close report a
// flush/close failure that would otherwise leave a silently truncated oracle.
func run() (err error) {
	outputDir := flag.String("out", defaultOutDir, "Output directory for the golden file")
	flag.Parse()

	// Fail fast on misinvocation: the default -out is module-relative, so a run
	// from the wrong working directory would scatter a stray oracle tree instead
	// of regenerating the real one. An explicit -out bypasses this guard.
	if *outputDir == defaultOutDir {
		if _, statErr := os.Stat("go.mod"); statErr != nil {
			return fmt.Errorf("default -out %q is module-relative; run from the repository root (no go.mod in working directory): %w", defaultOutDir, statErr)
		}
	}

	if mkErr := os.MkdirAll(*outputDir, 0o700); mkErr != nil {
		return fmt.Errorf("creating output directory: %w", mkErr)
	}

	filename := filepath.Join(*outputDir, "fibonacci_golden.json")
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- path comes from the -out flag (single-user build tool)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing output file: %w", cerr)
		}
	}()

	// Generate Fibonacci numbers
	// We'll generate a set of interesting cases:
	// - Small numbers (0-100)
	// - Powers of 2
	// - Powers of 10
	// - Random samples up to 10,000 (limit for reasonable file size/test time)

	// Targets cover three regimes:
	//   - Small: edge cases + early indices.
	//   - Medium: 1k..10k, schoolbook + Karatsuba.
	//   - Large (Audit-PRD E8-R1 / ADR-0004 §B5): exercise the FFT
	//     activation regime (default fftThreshold ≈ 115 kbits ≈ 35k
	//     decimal digits, so F(100k) and above sit in the FFT regime).
	targets := []uint64{
		0, 1, 2, 3, 4, 5, 10, 20, 50, 92, 93, 94, 100,
		128, 256, 512, 1000, 1024,
		2000, 2048, 5000, 8192, 10000,
		50_000, 100_000, 200_000,
	}

	data := make([]GoldenData, 0, len(targets))

	fmt.Println("Generating golden data...")

	for _, n := range targets {
		res := fibBig(n)
		data = append(data, GoldenData{
			N:      n,
			Result: res.String(),
		})
		fmt.Printf("Generated F(%d)\n", n)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if encErr := encoder.Encode(data); encErr != nil {
		return fmt.Errorf("encoding JSON: %w", encErr)
	}

	fmt.Printf("Successfully generated golden file at %s\n", filename)
	return nil
}

// fibBig calculates the nth Fibonacci number using math/big (iterative
// implementation). It serves as the ground-truth oracle for the golden
// test data in internal/fibonacci/testdata/fibonacci_golden.json.
//
// P2-04: this is an intentional duplicate of
// internal/fibonacci.calculateSmall. Do NOT unify them — see doc.go for
// the full rationale. Keeping two independent iterative kernels is what
// makes the golden tests meaningful: if calculator.go's calculateSmall
// ever silently regresses, the next regeneration cross-checks against
// this copy instead of merely re-confirming itself.
func fibBig(n uint64) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	if n == 1 {
		return big.NewInt(1)
	}

	a := big.NewInt(0)
	b := big.NewInt(1)

	for i := uint64(2); i <= n; i++ {
		// a, b = b, a+b
		a.Add(a, b) // a = a + b (temp result)
		a, b = b, a // swap: new a is old b, new b is sum
	}
	return b
}
