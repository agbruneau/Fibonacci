package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

func TestLUTCalculator(t *testing.T) {
	calc := &LUTCalculator{}
	ctx := context.Background()

	tests := []struct {
		n    uint64
		want *big.Int
	}{
		{0, big.NewInt(0)},
		{1, big.NewInt(1)},
		{2, big.NewInt(1)},
		{10, big.NewInt(55)},
		{93, lookupSmall(93)},
		{MaxLUTIndex, nil}, // Will verify value specifically
	}

	// Reference value for F(1024)
	f1024, _ := new(big.Int).SetString("4506699633677819813104383235728886049367860596218604830803023149600030645708721396248792609141030396244873266580345011219530209367425581019871067646094200262285202346655868899711089246778413354004103631553925405243", 10)
	tests[5].want = f1024

	for _, tt := range tests {
		got, err := calc.CalculateCore(ctx, nil, tt.n, Options{})
		if err != nil {
			t.Errorf("LUTCalculator.CalculateCore(%d) error = %v", tt.n, err)
			continue
		}
		if got.Cmp(tt.want) != 0 {
			t.Errorf("LUTCalculator.CalculateCore(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}

	// Test out of range
	_, err := calc.CalculateCore(ctx, nil, MaxLUTIndex+1, Options{})
	if err == nil {
		t.Errorf("LUTCalculator.CalculateCore(%d) expected error for out of range, got nil", MaxLUTIndex+1)
	}
}

func BenchmarkLUT(b *testing.B) {
	calc := &LUTCalculator{}
	ctx := context.Background()
	n := uint64(MaxLUTIndex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.CalculateCore(ctx, nil, n, Options{})
	}
}

func BenchmarkFastDoublingComparison(b *testing.B) {
	calc := &OptimizedFastDoubling{}
	ctx := context.Background()
	n := uint64(MaxLUTIndex)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = calc.CalculateCore(ctx, nil, n, Options{})
	}
}
