package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

func TestMockCalculator_Name(t *testing.T) {
	t.Parallel()

	mock := &MockCalculator{}
	name := mock.Name()

	if name != "mock" {
		t.Errorf("Name() = %q, want %q", name, "mock")
	}
}

func TestMockCalculator_Calculate(t *testing.T) {
	t.Parallel()

	t.Run("Calculate with Result", func(t *testing.T) {
		t.Parallel()
		expectedResult := big.NewInt(55)
		mock := &MockCalculator{
			Result: expectedResult,
			Err:    nil,
		}

		ctx := context.Background()
		result, err := mock.Calculate(ctx, nil, 0, 10, Options{})

		if err != nil {
			t.Errorf("Calculate() error = %v, want nil", err)
		}
		if result.Cmp(expectedResult) != 0 {
			t.Errorf("Calculate() result = %v, want %v", result, expectedResult)
		}
	})

	t.Run("Calculate with error", func(t *testing.T) {
		t.Parallel()
		expectedErr := &UnknownCalculatorError{Name: "test"}
		mock := &MockCalculator{
			Result: nil,
			Err:    expectedErr,
		}

		ctx := context.Background()
		result, err := mock.Calculate(ctx, nil, 0, 10, Options{})

		if err != expectedErr {
			t.Errorf("Calculate() error = %v, want %v", err, expectedErr)
		}
		if result != nil {
			t.Errorf("Calculate() result = %v, want nil", result)
		}
	})

	t.Run("Calculate with Fn", func(t *testing.T) {
		t.Parallel()
		expectedResult := big.NewInt(100)
		mock := &MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				return expectedResult, nil
			},
		}

		ctx := context.Background()
		result, err := mock.Calculate(ctx, nil, 0, 10, Options{})

		if err != nil {
			t.Errorf("Calculate() error = %v, want nil", err)
		}
		if result.Cmp(expectedResult) != 0 {
			t.Errorf("Calculate() result = %v, want %v", result, expectedResult)
		}
	})

	t.Run("Calculate with progress channel", func(t *testing.T) {
		t.Parallel()
		expectedResult := big.NewInt(55)
		progressChan := make(chan ProgressUpdate, 1)
		mock := &MockCalculator{
			Result: expectedResult,
			Err:    nil,
		}

		ctx := context.Background()
		result, err := mock.Calculate(ctx, progressChan, 0, 10, Options{})

		if err != nil {
			t.Errorf("Calculate() error = %v, want nil", err)
		}
		if result.Cmp(expectedResult) != 0 {
			t.Errorf("Calculate() result = %v, want %v", result, expectedResult)
		}

		// Check that progress was sent
		select {
		case update := <-progressChan:
			if update.Value != 1.0 {
				t.Errorf("Progress update value = %f, want 1.0", update.Value)
			}
		default:
			t.Error("Expected progress update to be sent")
		}
	})
}
