package metrics

import (
	"math/big"
	"strings"
	"testing"
)

// fibSmall computes F(n) via naive iteration; used only to build large
// *big.Int fixtures for lastNDigits.
func fibSmall(n int) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	a, b := big.NewInt(0), big.NewInt(1)
	for i := 2; i <= n; i++ {
		a.Add(a, b)
		a, b = b, a
	}
	return b
}

func TestDigitalRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		x    *big.Int
		want int
	}{
		{"zero", big.NewInt(0), 0},
		{"one", big.NewInt(1), 1},
		{"nine", big.NewInt(9), 9},
		{"ten", big.NewInt(10), 1},
		{"55 (F10)", big.NewInt(55), 1},   // 5+5=10, 1+0=1
		{"89 (F11)", big.NewInt(89), 8},   // 8+9=17, 1+7=8
		{"144 (F12)", big.NewInt(144), 9}, // 1+4+4=9
		{"233 (F13)", big.NewInt(233), 8}, // 2+3+3=8
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := digitalRoot(tt.x)
			if got != tt.want {
				t.Errorf("digitalRoot(%s) = %d, want %d", tt.x, got, tt.want)
			}
		})
	}
}

// TestLastNDigits covers lastNDigits, including APP-18: the zero-padding
// guard compared BitLen against n*4 bits/digit, but a decimal digit needs
// only log2(10) ≈ 3.32 bits. For n=20 that mismatch (67 bits actual vs 80
// bits required) meant 20-24 digit numbers whose last 20 digits start with
// zeros were not padded, silently truncating the reported LastDigits.
func TestLastNDigits(t *testing.T) {
	t.Parallel()
	f1000 := fibSmall(1000)
	f1000Str := f1000.String()
	pow20 := new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)

	tests := []struct {
		name string
		x    *big.Int
		n    int
		want string
	}{
		{"zero (non-positive guard)", big.NewInt(0), 20, "0"},
		{"F10 last 5", big.NewInt(55), 5, "55"},
		{"F12 last 3", big.NewInt(144), 3, "144"},
		{"F20 last 4", fibSmall(20), 4, "6765"}, // F(20) = 6765
		{"F1000 last 10", f1000, 10, f1000Str[len(f1000Str)-10:]},
		{"10^20 last 20 (zero padding)", pow20, 20, strings.Repeat("0", 20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := lastNDigits(tt.x, tt.n)
			if got != tt.want {
				t.Errorf("lastNDigits(%s, %d) = %q, want %q", tt.x, tt.n, got, tt.want)
			}
		})
	}
}
