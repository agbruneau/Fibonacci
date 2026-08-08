package fibonacci

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"testing"
)

func TestFastDoublingMod_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FastDoublingMod(ctx, 1_000_000, big.NewInt(1_000_000_007))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestFastDoublingMod_KnownValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		n    uint64
		mod  int64
		want int64
	}{
		{0, 1000, 0},
		{1, 1000, 1},
		{10, 1000, 55},
		{100, 10000, 5075},      // F(100) mod 10000 = 5075
		{1000, 1000000, 228875}, // F(1000) last 6 digits
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("N=%d_mod_%d", tc.n, tc.mod), func(t *testing.T) {
			t.Parallel()
			m := big.NewInt(tc.mod)
			result, err := FastDoublingMod(context.Background(), tc.n, m)
			if err != nil {
				t.Fatalf("FastDoublingMod error: %v", err)
			}
			if result.Int64() != tc.want {
				t.Errorf("FastDoublingMod(%d, %d) = %d, want %d",
					tc.n, tc.mod, result.Int64(), tc.want)
			}
		})
	}
}

func TestFastDoublingMod_ConsistentWithFull(t *testing.T) {
	t.Parallel()

	// Compute F(500) fully, then verify last 100 digits match modular result
	calc := MustNewCalculator(&FastDoublingCalculator{})
	ctx := context.Background()
	full, err := calc.Calculate(ctx, nil, 0, 500, Options{})
	if err != nil {
		t.Fatalf("full Calculate error: %v", err)
	}

	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil) // 10^100
	expected := new(big.Int).Mod(full, mod)

	result, err := FastDoublingMod(context.Background(), 500, mod)
	if err != nil {
		t.Fatalf("FastDoublingMod error: %v", err)
	}

	if result.Cmp(expected) != 0 {
		t.Errorf("modular result doesn't match full: got %s, want %s",
			result.String(), expected.String())
	}
}

func TestFastDoublingMod_InvalidModulus(t *testing.T) {
	t.Parallel()

	_, err := FastDoublingMod(context.Background(), 10, nil)
	if err == nil {
		t.Error("expected error for nil modulus")
	}

	_, err = FastDoublingMod(context.Background(), 10, big.NewInt(0))
	if err == nil {
		t.Error("expected error for zero modulus")
	}

	_, err = FastDoublingMod(context.Background(), 10, big.NewInt(-5))
	if err == nil {
		t.Error("expected error for negative modulus")
	}
}
