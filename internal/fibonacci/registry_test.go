package fibonacci

import (
	"context"
	"math/big"
	"testing"
)

// mockCoreCalculator is a simple implementation of coreCalculator for testing.
type mockCoreCalculator struct{}

func (m *mockCoreCalculator) Name() string { return "mock" }
func (m *mockCoreCalculator) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, opts Options) (*big.Int, error) {
	return big.NewInt(0), nil
}

func TestDefaultFactory(t *testing.T) {
	t.Parallel()
	factory := NewDefaultFactory()

	// Test Register and Has
	t.Run("RegisterAndHas", func(t *testing.T) {
		factory.Register("test", func() coreCalculator { return &mockCoreCalculator{} })
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

func TestGlobalFactory(t *testing.T) {
	t.Parallel()
	// Ensure GlobalFactory returns a non-nil factory
	f := GlobalFactory()
	if f == nil {
		t.Error("GlobalFactory returned nil")
	}

	// Ensure RegisterCalculator works
	RegisterCalculator("global_test", func() coreCalculator { return &mockCoreCalculator{} })
	if !f.Has("global_test") {
		t.Error("Global factory should have 'global_test' calculator")
	}
}
