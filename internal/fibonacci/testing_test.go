package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// TestNewTestFactory tests the TestFactory constructor.
func TestNewTestFactory(t *testing.T) {
	t.Run("nil calculators map", func(t *testing.T) {
		f := NewTestFactory(nil)
		if f == nil {
			t.Fatal("expected non-nil factory")
		}
		if len(f.List()) != 0 {
			t.Errorf("expected empty list, got %d items", len(f.List()))
		}
	})

	t.Run("with calculators", func(t *testing.T) {
		calcs := map[string]Calculator{
			"test": &mockCoreCalc{},
		}
		f := NewTestFactory(calcs)
		if len(f.List()) != 1 {
			t.Errorf("expected 1 calculator, got %d", len(f.List()))
		}
	})
}

// TestTestFactoryCreate tests the Create method.
func TestTestFactoryCreate(t *testing.T) {
	mock := &mockCoreCalc{}
	calcs := map[string]Calculator{
		"test": mock,
	}
	f := NewTestFactory(calcs)

	t.Run("existing calculator", func(t *testing.T) {
		calc, err := f.Create("test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calc != mock {
			t.Error("expected same calculator instance")
		}
	})

	t.Run("non-existing calculator", func(t *testing.T) {
		_, err := f.Create("unknown")
		if err == nil {
			t.Error("expected error for unknown calculator")
		}
	})
}

// TestTestFactoryGet tests the Get method.
func TestTestFactoryGet(t *testing.T) {
	mock := &mockCoreCalc{}
	calcs := map[string]Calculator{
		"test": mock,
	}
	f := NewTestFactory(calcs)

	calc, err := f.Get("test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calc != mock {
		t.Error("expected same calculator instance")
	}
}

// TestTestFactoryList tests the List method.
func TestTestFactoryList(t *testing.T) {
	calcs := map[string]Calculator{
		"alpha": &mockCoreCalc{},
		"beta":  &mockCoreCalc{},
		"gamma": &mockCoreCalc{},
	}
	f := NewTestFactory(calcs)

	list := f.List()
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
}

// TestTestFactoryRegister tests the Register method (no-op).
func TestTestFactoryRegister(t *testing.T) {
	f := NewTestFactory(nil)

	// Register is a no-op for TestFactory
	f.Register("test", func() coreCalculator { return &OptimizedFastDoubling{} })

	// Should still have no calculators since Register is no-op
	if len(f.List()) != 0 {
		t.Errorf("Register should be no-op, but list has %d items", len(f.List()))
	}
}

// TestTestFactoryGetAll tests the GetAll method.
func TestTestFactoryGetAll(t *testing.T) {
	mock1 := &mockCoreCalc{}
	mock2 := &mockCoreCalc{}
	calcs := map[string]Calculator{
		"test1": mock1,
		"test2": mock2,
	}
	f := NewTestFactory(calcs)

	all := f.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 calculators, got %d", len(all))
	}
	if all["test1"] != mock1 {
		t.Error("test1 not found in GetAll result")
	}
	if all["test2"] != mock2 {
		t.Error("test2 not found in GetAll result")
	}
}

// TestUnknownCalculatorError tests the error type.
func TestUnknownCalculatorError(t *testing.T) {
	err := &UnknownCalculatorError{Name: "myCalc"}
	expected := "unknown calculator: myCalc"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

// mockCoreCalc is a minimal implementation for testing.
type mockCoreCalc struct{}

func (m *mockCoreCalc) Name() string {
	return "mock"
}

func (m *mockCoreCalc) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, totalWork int, n uint64, opts Options) (*big.Int, error) {
	return big.NewInt(0), nil
}
