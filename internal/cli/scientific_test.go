package cli

import (
	"fmt"
	"math/big"
	"testing"
)

// TestScientificNotation_MatchesBigFloat pins scientificNotation to the
// rendering fmt's %.6e gives for the exact integer, on the cases where the two
// could diverge: exact ties (half-to-even both ways), carries through a run of
// nines into a new decade, values with exactly seven digits, and a real F(n).
func TestScientificNotation_MatchesBigFloat(t *testing.T) {
	t.Parallel()
	fib := big.NewInt(0)
	b := big.NewInt(1)
	for i := 0; i < 5000; i++ {
		fib.Add(fib, b)
		fib, b = b, fib
	}
	cases := []string{
		"1234567", "9999999", "10000000", "12345678", "12345675", "12345665",
		"12345675000000000001", "99999995", "99999994999", "10000005", "12345670",
		"3141592653589793238462643383279502884197", fib.String(),
	}

	for _, digits := range cases {
		x, ok := new(big.Int).SetString(digits, 10)
		if !ok {
			t.Fatalf("bad test digits %q", digits)
		}
		want := fmt.Sprintf("%.6e", new(big.Float).SetInt(x))
		if got := scientificNotation(digits); got != want {
			t.Errorf("scientificNotation(%s) = %s, want %s", digits, got, want)
		}
	}
}
