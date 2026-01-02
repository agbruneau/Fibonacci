package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// mockCalculator implements fibonacci.Calculator for testing.
type mockCalculator struct {
	name   string
	result *big.Int
	err    error
}

func (m *mockCalculator) Name() string {
	return m.name
}

func (m *mockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, totalWork int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return new(big.Int).Set(m.result), nil
	}
	// Return a simple Fibonacci result for small n
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}
	return big.NewInt(int64(n)), nil // Simplified for testing
}

// TestNewCalculatorService tests the constructor.
func TestNewCalculatorService(t *testing.T) {
	factory := fibonacci.NewTestFactory(make(map[string]fibonacci.Calculator))
	cfg := config.AppConfig{
		Threshold:         4096,
		FFTThreshold:      500000,
		StrassenThreshold: 3072,
	}

	svc := NewCalculatorService(factory, cfg, 1000000)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.factory == nil {
		t.Error("factory should not be nil")
	}
	if svc.maxN != 1000000 {
		t.Errorf("expected maxN 1000000, got %d", svc.maxN)
	}
}

// TestCalculate tests the Calculate method.
func TestCalculate(t *testing.T) {
	tests := []struct {
		name        string
		algoName    string
		n           uint64
		maxN        uint64
		setupCalc   func() *mockCalculator
		expectError bool
		expectValue int64
	}{
		{
			name:     "successful calculation",
			algoName: "fast",
			n:        10,
			maxN:     100,
			setupCalc: func() *mockCalculator {
				return &mockCalculator{name: "fast", result: big.NewInt(55)}
			},
			expectError: false,
			expectValue: 55,
		},
		{
			name:        "exceeds max n",
			algoName:    "fast",
			n:           200,
			maxN:        100,
			setupCalc:   nil,
			expectError: true,
		},
		{
			name:     "max n is zero (no limit)",
			algoName: "fast",
			n:        1000000,
			maxN:     0,
			setupCalc: func() *mockCalculator {
				return &mockCalculator{name: "fast", result: big.NewInt(12345)}
			},
			expectError: false,
			expectValue: 12345,
		},
		{
			name:        "algorithm not found",
			algoName:    "unknown",
			n:           10,
			maxN:        100,
			setupCalc:   nil,
			expectError: true,
		},
		{
			name:     "calculation error",
			algoName: "fast",
			n:        10,
			maxN:     100,
			setupCalc: func() *mockCalculator {
				return &mockCalculator{name: "fast", err: errors.New("calculation failed")}
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calcs := make(map[string]fibonacci.Calculator)
			if tc.setupCalc != nil {
				calc := tc.setupCalc()
				calcs[tc.algoName] = calc
			}
			factory := fibonacci.NewTestFactory(calcs)

			cfg := config.AppConfig{
				Threshold:         4096,
				FFTThreshold:      500000,
				StrassenThreshold: 3072,
			}
			svc := NewCalculatorService(factory, cfg, tc.maxN)

			result, err := svc.Calculate(context.Background(), tc.algoName, tc.n)

			if tc.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Int64() != tc.expectValue {
				t.Errorf("expected %d, got %d", tc.expectValue, result.Int64())
			}
		})
	}
}

// TestCalculateWithContext tests that context cancellation works.
func TestCalculateWithContext(t *testing.T) {
	factory := fibonacci.NewTestFactory(map[string]fibonacci.Calculator{
		"fast": &mockCalculator{name: "fast", result: big.NewInt(55)},
	})

	cfg := config.AppConfig{}
	svc := NewCalculatorService(factory, cfg, 0)

	// Use a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The mock doesn't actually check context, so this just tests the plumbing
	result, err := svc.Calculate(ctx, "fast", 10)
	// Since our mock doesn't check context, it should still succeed
	if err != nil {
		t.Logf("Got error (may be expected with context cancellation): %v", err)
	}
	if result != nil {
		t.Logf("Got result: %v", result)
	}
}

// TestErrMaxValueExceeded tests the error variable.
func TestErrMaxValueExceeded(t *testing.T) {
	if ErrMaxValueExceeded.Error() != "maximum n value exceeded" {
		t.Errorf("unexpected error message: %s", ErrMaxValueExceeded.Error())
	}
}

// TestServiceInterface tests that CalculatorService implements Service interface.
func TestServiceInterface(t *testing.T) {
	var _ Service = (*CalculatorService)(nil)
}
