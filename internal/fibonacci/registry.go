package fibonacci

// Note: CalculatorFactory interface is not mockable with mockgen because Register()
// uses the unexported coreCalculator type. Use DefaultFactory or manual mocks instead.

import (
	"fmt"
	"sort"
	"sync"
)

// CalculatorFactory is an interface for creating Calculator instances.
// It allows for flexible calculator instantiation and registration,
// enabling dependency injection and easier testing.
type CalculatorFactory interface {
	// Create creates a new Calculator instance by name.
	// Returns an error if the calculator type is not registered.
	Create(name string) (Calculator, error)

	// Get returns an existing Calculator instance by name.
	// Returns an error if the calculator type is not registered.
	Get(name string) (Calculator, error)

	// List returns a sorted list of registered calculator names.
	List() []string

	// Register adds a new calculator type to the factory.
	Register(name string, creator func() coreCalculator) error

	// GetAll returns a map of all registered calculators.
	GetAll() map[string]Calculator
}

// DefaultFactory is the default implementation of CalculatorFactory.
// It maintains a thread-safe registry of calculator creators and
// caches Calculator instances for reuse.
type DefaultFactory struct {
	mu          sync.RWMutex
	creators    map[string]func() coreCalculator
	calculators map[string]Calculator
}

// NewDefaultFactory creates a new DefaultFactory with the standard
// Fibonacci calculator implementations pre-registered.
//
// Pre-registered calculators:
//   - "fast": OptimizedFastDoubling (O(log n), Parallel, Zero-Alloc)
//   - "matrix": MatrixExponentiation (O(log n), Parallel, Zero-Alloc)
//   - "fft": FFTBasedCalculator (O(log n), FFT-accelerated)
//
// Returns:
//   - *DefaultFactory: A new factory with default calculators registered.
func NewDefaultFactory() *DefaultFactory {
	f := &DefaultFactory{
		creators:    make(map[string]func() coreCalculator),
		calculators: make(map[string]Calculator),
	}

	// Register the default calculators
	_ = f.Register("fast", func() coreCalculator { return &OptimizedFastDoubling{} })
	_ = f.Register("matrix", func() coreCalculator { return &MatrixExponentiation{} })
	_ = f.Register("fft", func() coreCalculator { return &FFTBasedCalculator{} })

	return f
}

// Register adds a new calculator type to the factory.
// The creator function is called lazily when the calculator is first requested.
// If a calculator with the same name already exists, it will be replaced.
//
// Parameters:
//   - name: The unique identifier for the calculator type.
//   - creator: A function that creates a new coreCalculator instance.
func (f *DefaultFactory) Register(name string, creator func() coreCalculator) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[name] = creator
	// Clear cached calculator if it exists, so it will be recreated with the new creator
	delete(f.calculators, name)
	return nil
}

// Create creates a new Calculator instance by name.
// Unlike Get(), this always creates a fresh instance without caching.
//
// Parameters:
//   - name: The name of the calculator type to create.
//
// Returns:
//   - Calculator: A new Calculator instance.
//   - error: An error if the calculator type is not registered.
func (f *DefaultFactory) Create(name string) (Calculator, error) {
	f.mu.RLock()
	creator, ok := f.creators[name]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown calculator: %s", name)
	}
	return NewCalculator(creator()), nil
}

// Get returns a Calculator instance by name.
// Instances are cached and reused for subsequent calls with the same name.
// This is the preferred method for most use cases.
//
// Parameters:
//   - name: The name of the calculator to retrieve.
//
// Returns:
//   - Calculator: The Calculator instance.
//   - error: An error if the calculator type is not registered.
func (f *DefaultFactory) Get(name string) (Calculator, error) {
	// Check cache first with read lock
	f.mu.RLock()
	if calc, exists := f.calculators[name]; exists {
		f.mu.RUnlock()
		return calc, nil
	}
	f.mu.RUnlock()

	// Create new calculator with write lock
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if calc, exists := f.calculators[name]; exists {
		return calc, nil
	}

	creator, ok := f.creators[name]
	if !ok {
		return nil, fmt.Errorf("unknown calculator: %s", name)
	}

	calc := NewCalculator(creator())
	f.calculators[name] = calc
	return calc, nil
}

// List returns a sorted list of all registered calculator names.
// The list is sorted alphabetically for consistent ordering.
//
// Returns:
//   - []string: A sorted slice of calculator names.
func (f *DefaultFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.creators))
	for name := range f.creators {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAll returns a map of all registered calculators.
// This method lazily initializes all calculators that haven't been
// created yet.
//
// Returns:
//   - map[string]Calculator: A map of calculator names to Calculator instances.
func (f *DefaultFactory) GetAll() map[string]Calculator {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Ensure all calculators are initialized
	for name, creator := range f.creators {
		if _, exists := f.calculators[name]; !exists {
			f.calculators[name] = NewCalculator(creator())
		}
	}

	// Return a copy to prevent external modifications
	result := make(map[string]Calculator, len(f.calculators))
	for name, calc := range f.calculators {
		result[name] = calc
	}
	return result
}

// MustGet is like Get but panics if the calculator is not found.
// This is useful in initialization code where missing calculators
// should be considered a programming error.
//
// Parameters:
//   - name: The name of the calculator to retrieve.
//
// Returns:
//   - Calculator: The Calculator instance.
//
// Panics:
//   - If the calculator type is not registered.
func (f *DefaultFactory) MustGet(name string) Calculator {
	calc, err := f.Get(name)
	if err != nil {
		panic(fmt.Sprintf("fibonacci: required calculator not found: %s", name))
	}
	return calc
}

// Has checks if a calculator with the given name is registered.
//
// Parameters:
//   - name: The name of the calculator to check.
//
// Returns:
//   - bool: true if the calculator is registered, false otherwise.
func (f *DefaultFactory) Has(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, exists := f.creators[name]
	return exists
}

// globalFactory is the default global factory instance.
var globalFactory = NewDefaultFactory()

// GlobalFactory returns the global factory instance.
// This is a convenience for applications that don't need
// multiple factory instances.
//
// Returns:
//   - *DefaultFactory: The global factory instance.
func GlobalFactory() *DefaultFactory {
	return globalFactory
}

// RegisterCalculator registers a calculator in the global factory.
// This is a convenience function for adding custom calculators.
//
// Parameters:
//   - name: The unique identifier for the calculator type.
//   - creator: A function that creates a new coreCalculator instance.
func RegisterCalculator(name string, creator func() coreCalculator) error {
	return globalFactory.Register(name, creator)
}
