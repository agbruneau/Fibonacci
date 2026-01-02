package fibonacci

import (
	"context"
	"math/big"
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
func (m *MockCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, opts Options) (*big.Int, error) {
	if m.Fn != nil {
		return m.Fn(ctx, n)
	}
	if progressChan != nil {
		progressChan <- ProgressUpdate{CalculatorIndex: calcIndex, Value: 1.0}
	}
	return m.Result, m.Err
}

// TestFactory is a CalculatorFactory implementation designed for testing.
// It allows tests in other packages to create factories with mock calculators.
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

// Create returns the calculator by name.
func (f *TestFactory) Create(name string) (Calculator, error) {
	return f.Get(name)
}

// Get returns the calculator by name.
func (f *TestFactory) Get(name string) (Calculator, error) {
	calc, ok := f.calculators[name]
	if !ok {
		return nil, &UnknownCalculatorError{Name: name}
	}
	return calc, nil
}

// List returns all registered calculator names.
func (f *TestFactory) List() []string {
	names := make([]string, 0, len(f.calculators))
	for name := range f.calculators {
		names = append(names, name)
	}
	return names
}

// Register is a no-op for TestFactory as calculators are provided at construction.
func (f *TestFactory) Register(name string, creator func() coreCalculator) error {
	// No-op: calculators are set at construction time
	return nil
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
