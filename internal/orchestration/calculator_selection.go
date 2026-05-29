package orchestration

import (
	"github.com/agbruneau/FibGo/internal/fibonacci"
)

// GetCalculatorsToRun determines which calculators should be executed based on
// the algorithm name. Returns calculators in alphabetically sorted order for
// consistent, reproducible behavior.
//
// Parameters:
//   - algo: The algorithm name ("fast", "matrix", "fft", "all").
//   - factory: The calculator factory to retrieve implementations from.
//
// Returns:
//   - []fibonacci.Calculator: A slice of calculators to execute.
func GetCalculatorsToRun(algo string, factory fibonacci.CalculatorFactory) []fibonacci.Calculator {
	if algo == "all" {
		keys := factory.List() // List() returns sorted keys
		calculators := make([]fibonacci.Calculator, 0, len(keys))
		for _, k := range keys {
			if calc, err := factory.Get(k); err == nil {
				calculators = append(calculators, calc)
			}
		}
		return calculators
	}
	if calc, err := factory.Get(algo); err == nil {
		return []fibonacci.Calculator{calc}
	}
	return nil
}
