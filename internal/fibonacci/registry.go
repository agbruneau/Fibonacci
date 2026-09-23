package fibonacci

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// DefaultFactory is the calculator registry the binary uses. It maintains a
// thread-safe registry of calculator creators and caches Calculator instances
// for reuse.
//
// It implements no interface declared here: the consumers define what they need
// (orchestration.CalculatorSource, app.CalculatorRegistry) and this type
// satisfies them structurally. The five-method CalculatorFactory that used to
// live in this file was provider-defined and wider than any caller
// (audit API-01).
type DefaultFactory struct {
	mu          sync.RWMutex
	creators    map[string]func() CoreCalculator
	calculators map[string]Calculator
}

// taggedRegistrations lists the calculators that exist only under a build
// tag. Each tagged file appends its registration from init(), so every factory
// NewDefaultFactory builds carries it; the default build leaves it empty. This
// is what makes `-algo gmp` reachable with `-tags gmp` (EVAL-23).
var taggedRegistrations []func(*DefaultFactory)

// NewDefaultFactory creates a new DefaultFactory with the standard
// Fibonacci calculator implementations pre-registered.
//
// Pre-registered calculators:
//   - "fast": FastDoublingCalculator (O(log n), Parallel, Zero-Alloc)
//   - "matrix": MatrixExponentiationCalculator (O(log n), Parallel, Zero-Alloc)
//   - "fft": FFTBasedCalculator (O(log n), FFT-accelerated)
//   - "gmp": GMPCalculator, only in a build with the gmp tag
//
// Returns:
//   - *DefaultFactory: A new factory with default calculators registered.
func NewDefaultFactory() *DefaultFactory {
	f := &DefaultFactory{
		creators:    make(map[string]func() CoreCalculator),
		calculators: make(map[string]Calculator),
	}

	// Register the built-in calculators. Discarding these errors used to be
	// safe only because Register could not fail; now that it validates, a
	// failure here means this very function is wrong — a duplicate name or a
	// nil creator literal — which is a programmer bug, not a runtime
	// condition. Same contract as MustNewCalculator.
	builtins := []struct {
		name    string
		creator func() CoreCalculator
	}{
		{"fast", func() CoreCalculator { return &FastDoublingCalculator{} }},
		{"matrix", func() CoreCalculator { return &MatrixExponentiationCalculator{} }},
		{"fft", func() CoreCalculator { return &FFTBasedCalculator{} }},
	}
	for _, b := range builtins {
		if err := f.Register(b.name, b.creator); err != nil {
			panic(fmt.Sprintf("fibonacci: registering built-in calculator %q: %v", b.name, err))
		}
	}
	for _, register := range taggedRegistrations {
		register(f)
	}

	return f
}

// Register adds a new calculator type to the factory. The creator function is
// called lazily, when the calculator is first requested.
//
// It returns an error for an empty name, a nil creator, or a name that is
// already registered.
//
// Until audit API-02 this method could not fail: it returned nil
// unconditionally, accepted a nil creator (deferring the panic to the first
// Get), and REPLACED an existing registration in silence — so a typo'd or
// duplicated name quietly swapped out a built-in calculator. Callers were
// nonetheless obliged to check the error by errcheck, which made the check pure
// ceremony. The signature now means something.
//
// Refusing a duplicate rather than replacing is the deliberate half of this: a
// factory has three built-ins registered by NewDefaultFactory, and nothing in
// the code base re-registers a name on purpose. A caller that genuinely wants a
// different implementation should build its own factory.
//
// Parameters:
//   - name: The unique identifier for the calculator type.
//   - creator: A function that creates a new CoreCalculator instance.
func (f *DefaultFactory) Register(name string, creator func() CoreCalculator) error {
	if name == "" {
		return errors.New("fibonacci: calculator name must not be empty")
	}
	if creator == nil {
		return fmt.Errorf("fibonacci: calculator %q: creator must not be nil", name)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.creators[name]; exists {
		return fmt.Errorf("fibonacci: calculator %q is already registered", name)
	}
	f.creators[name] = creator
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
	calc, err := NewCalculator(creator())
	if err != nil {
		return nil, err
	}
	return calc, nil
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

	calc, err := NewCalculator(creator())
	if err != nil {
		return nil, err
	}
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
// It has no error return, so an entry whose creator misbehaves is omitted from
// the result rather than failing the call. Since Register rejects a nil creator
// (audit API-02), the only way to land here is a registered creator that
// returns a nil CoreCalculator — a bug in that creator. The omission is
// visible: the caller (auto-calibration) reports "no usable profile" when the
// calculator it needs is missing, instead of the map silently under-reporting
// as it did when Register accepted anything.
//
// Returns:
//   - map[string]Calculator: A map of calculator names to Calculator instances.
func (f *DefaultFactory) GetAll() map[string]Calculator {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Ensure all calculators are initialized
	for name, creator := range f.creators {
		if _, exists := f.calculators[name]; !exists {
			if calc, err := NewCalculator(creator()); err == nil {
				f.calculators[name] = calc
			}
		}
	}

	// Return a copy to prevent external modifications
	result := make(map[string]Calculator, len(f.calculators))
	for name, calc := range f.calculators {
		result[name] = calc
	}
	return result
}
