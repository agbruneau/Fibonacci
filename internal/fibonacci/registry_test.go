package fibonacci

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/agbruneau/FibGo/internal/progress"
)

// mockCoreCalculator is a simple implementation of CoreCalculator for testing.
type mockCoreCalculator struct{}

func (m *mockCoreCalculator) Name() string { return "mock" }
func (m *mockCoreCalculator) CalculateCore(ctx context.Context, reporter progress.ProgressCallback, n uint64, opts Options) (*big.Int, error) {
	return big.NewInt(0), nil
}

func TestDefaultFactory(t *testing.T) {
	t.Parallel()
	factory := NewDefaultFactory()

	// Test Register and Has
	t.Run("RegisterAndHas", func(t *testing.T) {
		factory.Register("test", func() CoreCalculator { return &mockCoreCalculator{} })
		if !factory.Has("test") {
			t.Error("Factory should have 'test' calculator")
		}
		if factory.Has("nonexistent") {
			t.Error("Factory should not have 'nonexistent' calculator")
		}
	})

	// Test GetAll
	t.Run("GetAll", func(t *testing.T) {
		calculators := factory.GetAll()
		if len(calculators) < 1 { // Should have at least the default ones + "test"
			t.Error("GetAll should return calculators")
		}
		if _, ok := calculators["test"]; !ok {
			t.Error("GetAll should contain 'test' calculator")
		}
	})

	// Test Create
	t.Run("Create", func(t *testing.T) {
		calc, err := factory.Create("test")
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
		if calc == nil {
			t.Error("Create returned nil calculator")
		}
		_, err = factory.Create("nonexistent")
		if err == nil {
			t.Error("Create should fail for nonexistent calculator")
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		// First call creates
		calc1, err := factory.Get("test")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}

		// Second call returns cached
		calc2, err := factory.Get("test")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}

		if calc1 != calc2 {
			t.Error("Get should return cached instance")
		}

		_, err = factory.Get("nonexistent")
		if err == nil {
			t.Error("Get should fail for nonexistent calculator")
		}
	})

	// Test MustGet
	t.Run("MustGet", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// panic expected for nonexistent
			}
		}()
		_ = factory.MustGet("test")
		// This should panic
		_ = factory.MustGet("nonexistent")
		t.Error("MustGet should have panicked for nonexistent calculator")
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		list := factory.List()
		found := false
		for _, name := range list {
			if name == "test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("List should contain 'test'")
		}
	})
}

// TestCalculatorFactory_ConcurrentCreation verifies that DefaultFactory.Create
// is safe for concurrent use. 10 goroutines call Create("fast") simultaneously
// on a shared factory and all must receive non-nil calculators without panics.
func TestCalculatorFactory_ConcurrentCreation(t *testing.T) {
	t.Parallel()

	factory := NewDefaultFactory()
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	calculators := make([]Calculator, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			calculators[idx], errs[idx] = factory.Create("fast")
		}(i)
	}

	wg.Wait()

	for i := range goroutines {
		if errs[i] != nil {
			t.Errorf("goroutine %d: Create returned error: %v", i, errs[i])
		}
		if calculators[i] == nil {
			t.Errorf("goroutine %d: Create returned nil calculator", i)
		}
	}
}
