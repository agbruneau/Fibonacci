package fibonacci

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
func (f *TestFactory) Register(name string, creator func() coreCalculator) {
	// No-op: calculators are set at construction time
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

