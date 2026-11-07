package fibonacci

import (
	"context"
	"math/big"
	"sort"

	"github.com/agbruneau/FibGo/internal/progress"
)

// MockCalculator is a mock implementation of the Calculator interface.
// It is exported to allow external packages (like cmd/fibcalc) to use it for testing.
type MockCalculator struct {
	Result *big.Int
	Err    error
	Fn     func(ctx context.Context, n uint64) (*big.Int, error)
}

// Name returns the calculator name.
func (m *MockCalculator) Name() string {
	return "mock"
}

// Calculate returns the pre-configured Result and Err, or calls Fn if provided.
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- progress.ProgressUpdate, calcIndex int, n uint64, opts Options) (*big.Int, error) {
	if m.Fn != nil {
		return m.Fn(ctx, n)
	}
	if progressChan != nil {
		progressChan <- progress.ProgressUpdate{CalculatorIndex: calcIndex, Value: 1.0}
	}
	return m.Result, m.Err
}

// TestFactory is a calculator registry for tests: it satisfies
// orchestration.CalculatorSource and app.CalculatorRegistry with a fixed set of
// calculators supplied at construction.
//
// It no longer carries Register or Create. Both existed only to satisfy the
// five-method fibonacci.CalculatorFactory interface (audit API-01), and Register
// was a no-op that returned nil — a double claiming to accept a registration it
// discarded. With the interface gone, so are they.
type TestFactory struct {
	calculators map[string]Calculator
}

// NewTestFactory creates a factory pre-populated with the given calculators.
// This is intended for use in tests where mock calculators are needed.
//
// Parameters:
//   - calculators: A map of calculator names to Calculator instances.
//
// Returns:
//   - *TestFactory: A factory that can be used in place of DefaultFactory in tests.
func NewTestFactory(calculators map[string]Calculator) *TestFactory {
	if calculators == nil {
		calculators = make(map[string]Calculator)
	}
	return &TestFactory{calculators: calculators}
}

// Get returns the calculator by name.
func (f *TestFactory) Get(name string) (Calculator, error) {
	calc, ok := f.calculators[name]
	if !ok {
		return nil, &UnknownCalculatorError{Name: name}
	}
	return calc, nil
}

// List returns a sorted list of all registered calculator names.
//
// Sorted, like DefaultFactory.List and as the CalculatorFactory contract states
// (audit API-01). A fake that claims to be interchangeable with the real
// factory but returns Go's randomized map-iteration order makes any test
// asserting on ordered output — shell completion, the execution-mode banner —
// pass or fail on map iteration order rather than on behavior.
func (f *TestFactory) List() []string {
	names := make([]string, 0, len(f.calculators))
	for name := range f.calculators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAll returns all calculators.
func (f *TestFactory) GetAll() map[string]Calculator {
	result := make(map[string]Calculator, len(f.calculators))
	for k, v := range f.calculators {
		result[k] = v
	}
	return result
}

// UnknownCalculatorError is returned when a calculator name is not found.
type UnknownCalculatorError struct {
	Name string
}

func (e *UnknownCalculatorError) Error() string {
	return "unknown calculator: " + e.Name
}
