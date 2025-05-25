package main

import (
	"context"
	"math/big"
	"testing"
)

// fibTestCases defines the shared test cases for Fibonacci functions.
var fibTestCases = []struct {
	name     string
	n        int
	expected *big.Int
}{
	{"F(0)", 0, big.NewInt(0)},
	{"F(1)", 1, big.NewInt(1)},
	{"F(2)", 2, big.NewInt(1)},
	{"F(10)", 10, big.NewInt(55)},
	{"F(20)", 20, big.NewInt(6765)},
	{"F(30)", 30, big.NewInt(832040)}, // Added a slightly larger case
	{"F(70)", 70, func() *big.Int { v, _ := new(big.Int).SetString("190392490709135", 10); return v }()},
}

func TestFibBinet(t *testing.T) {
	pool := newIntPool()
	for _, tc := range fibTestCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Mark as parallelizable
			result, err := fibBinet(context.Background(), nil, tc.n, pool)
			if err != nil {
				t.Fatalf("fibBinet(%d) returned an unexpected error: %v", tc.n, err)
			}
			if result.Cmp(tc.expected) != 0 {
				t.Errorf("fibBinet(%d) = %v, want %v", tc.n, result, tc.expected)
			}
		})
	}
}

func TestFibFastDoubling(t *testing.T) {
	pool := newIntPool()
	for _, tc := range fibTestCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Mark as parallelizable
			result, err := fibFastDoubling(context.Background(), nil, tc.n, pool)
			if err != nil {
				t.Fatalf("fibFastDoubling(%d) returned an unexpected error: %v", tc.n, err)
			}
			if result.Cmp(tc.expected) != 0 {
				t.Errorf("fibFastDoubling(%d) = %v, want %v", tc.n, result, tc.expected)
			}
		})
	}
}

func TestFibMatrix(t *testing.T) {
	pool := newIntPool()
	for _, tc := range fibTestCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Mark as parallelizable
			result, err := fibMatrix(context.Background(), nil, tc.n, pool)
			if err != nil {
				t.Fatalf("fibMatrix(%d) returned an unexpected error: %v", tc.n, err)
			}
			if result.Cmp(tc.expected) != 0 {
				t.Errorf("fibMatrix(%d) = %v, want %v", tc.n, result, tc.expected)
			}
		})
	}
}

// Test for negative input, which should be handled gracefully by all functions.
func TestFibNegativeInput(t *testing.T) {
	pool := newIntPool()
	algorithms := []struct {
		name string
		fn   fibFunc
	}{
		{"Binet", fibBinet},
		{"FastDoubling", fibFastDoubling},
		{"Matrix", fibMatrix},
	}

	for _, algo := range algorithms {
		algo := algo // capture range variable
		t.Run(algo.name, func(t *testing.T) {
			t.Parallel() // Mark as parallelizable
			_, err := algo.fn(context.Background(), nil, -1, pool)
			if err == nil {
				t.Errorf("%s(-1) expected an error for negative input, but got nil", algo.name)
			}
		})
	}
}
