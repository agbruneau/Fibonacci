package fibonacci

import (
	"context"
	"errors"
	"math/big"
	"sort"
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

// TestTestFactory_CreateAndGet covers Create (P2-06). NewTestFactory is
// exercised indirectly by every subtest here.
func TestTestFactory_CreateAndGet(t *testing.T) {
	t.Parallel()

	mock := &MockCalculator{Result: big.NewInt(55)}
	f := NewTestFactory(map[string]Calculator{"mock": mock})

	got, err := f.Create("mock")
	if err != nil {
		t.Fatalf("Create(mock): unexpected error %v", err)
	}
	if got != mock {
		t.Errorf("Create returned unexpected calculator: %#v", got)
	}

	got, err = f.Get("mock")
	if err != nil {
		t.Fatalf("Get(mock): unexpected error %v", err)
	}
	if got != mock {
		t.Errorf("Get returned unexpected calculator: %#v", got)
	}
}

// TestTestFactory_Create_Unknown covers the error path of Create/Get and
// the UnknownCalculatorError.Error method.
func TestTestFactory_Create_Unknown(t *testing.T) {
	t.Parallel()

	f := NewTestFactory(nil)
	_, err := f.Create("nope")
	if err == nil {
		t.Fatal("expected UnknownCalculatorError from Create on unknown name")
	}

	var uce *UnknownCalculatorError
	if !errors.As(err, &uce) {
		t.Fatalf("expected *UnknownCalculatorError, got %T", err)
	}
	if uce.Name != "nope" {
		t.Errorf("UnknownCalculatorError.Name: want %q, got %q", "nope", uce.Name)
	}
	const want = "unknown calculator: nope"
	if uce.Error() != want {
		t.Errorf("UnknownCalculatorError.Error() = %q, want %q", uce.Error(), want)
	}
}

// TestTestFactory_NewTestFactory_NilMap ensures a nil map is tolerated and
// yields an empty registry.
func TestTestFactory_NewTestFactory_NilMap(t *testing.T) {
	t.Parallel()

	f := NewTestFactory(nil)
	if f == nil {
		t.Fatal("NewTestFactory(nil) returned nil")
	}
	if got := f.List(); len(got) != 0 {
		t.Errorf("expected empty calculator list for nil map, got %v", got)
	}
}

// TestTestFactory_List covers List.
func TestTestFactory_List(t *testing.T) {
	t.Parallel()

	calcs := map[string]Calculator{
		"alpha": &MockCalculator{Result: big.NewInt(1)},
		"beta":  &MockCalculator{Result: big.NewInt(2)},
		"gamma": &MockCalculator{Result: big.NewInt(3)},
	}
	f := NewTestFactory(calcs)

	names := f.List()
	sort.Strings(names)
	want := []string{"alpha", "beta", "gamma"}
	if len(names) != len(want) {
		t.Fatalf("List length: want %d, got %d (%v)", len(want), len(names), names)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("List[%d]: want %q, got %q", i, w, names[i])
		}
	}
}

// TestTestFactory_Register_IsNoOp verifies the documented no-op behaviour.
func TestTestFactory_Register_IsNoOp(t *testing.T) {
	t.Parallel()

	f := NewTestFactory(map[string]Calculator{"mock": &MockCalculator{}})

	err := f.Register("other", func() CoreCalculator { return nil })
	if err != nil {
		t.Fatalf("Register returned unexpected error: %v", err)
	}

	if _, err := f.Get("other"); err == nil {
		t.Error("expected 'other' to remain unknown after Register no-op")
	}
	if _, err := f.Get("mock"); err != nil {
		t.Errorf("expected 'mock' to remain known, got error: %v", err)
	}
}

// TestTestFactory_GetAll covers GetAll and verifies the returned map is a
// defensive copy (mutating it does not mutate the factory).
func TestTestFactory_GetAll(t *testing.T) {
	t.Parallel()

	m1 := &MockCalculator{Result: big.NewInt(1)}
	m2 := &MockCalculator{Result: big.NewInt(2)}
	f := NewTestFactory(map[string]Calculator{"one": m1, "two": m2})

	got := f.GetAll()
	if len(got) != 2 {
		t.Fatalf("GetAll: want 2 entries, got %d", len(got))
	}
	if got["one"] != m1 {
		t.Errorf("GetAll[one] mismatch")
	}
	if got["two"] != m2 {
		t.Errorf("GetAll[two] mismatch")
	}

	// GetAll must return a copy — mutating it must not affect the factory.
	delete(got, "one")
	if _, err := f.Get("one"); err != nil {
		t.Errorf("mutating GetAll result leaked back into factory: %v", err)
	}
}
