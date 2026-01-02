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

func main() {
	outputDir := flag.String("out", "internal/fibonacci/testdata", "Output directory for the golden file")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	filename := filepath.Join(*outputDir, "fibonacci_golden.json")
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Generate Fibonacci numbers
	// We'll generate a set of interesting cases:
	// - Small numbers (0-100)
	// - Powers of 2
	// - Powers of 10
	// - Random samples up to 10,000 (limit for reasonable file size/test time)

	targets := []uint64{
		0, 1, 2, 3, 4, 5, 10, 20, 50, 92, 93, 94, 100,
		128, 256, 512, 1000, 1024,
		2000, 2048, 5000, 8192, 10000,
	}

	var data []GoldenData

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
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated golden file at %s\n", filename)
}

// fibBig calculates the nth Fibonacci number using math/big (iterative implementation)
// This serves as our "Oracle" using the standard library.
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
