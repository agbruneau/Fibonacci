//go:build gmp

package fibonacci

import (
	"context"
	"testing"
)

func TestGMPCalculator_CalculateCore(t *testing.T) {
	calc := &GMPCalculator{}
	ctx := context.Background()
	noopReporter := func(float64) {}
	opts := Options{}

	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{2, "1"},
		{3, "2"},
		{4, "3"},
		{5, "5"},
		{10, "55"},
		{20, "6765"},
		{50, "12586269025"},
	}

	for _, tt := range tests {
		got, err := calc.CalculateCore(ctx, noopReporter, tt.n, opts)
		if err != nil {
			t.Errorf("CalculateCore(%d) error = %v", tt.n, err)
			continue
		}
		if got.String() != tt.want {
			t.Errorf("CalculateCore(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestGMPCalculator_CalculateCore_Cancel(t *testing.T) {
	calc := &GMPCalculator{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	noopReporter := func(float64) {}
	opts := Options{}

	_, err := calc.CalculateCore(ctx, noopReporter, 1000, opts)
	if err == nil {
		t.Error("CalculateCore(canceled context) expected error, got nil")
	}
}

func TestGMPCalculator_Name(t *testing.T) {
	calc := &GMPCalculator{}
	if calc.Name() != "GMP (Fast Doubling)" {
		t.Errorf("Name() = %v, want %v", calc.Name(), "GMP (Fast Doubling)")
	}
}
