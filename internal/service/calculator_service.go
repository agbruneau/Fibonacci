package service

import (
	"context"
	"errors"
	"math/big"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

var (
	// ErrMaxValueExceeded is returned when n exceeds the configured maximum limit.
	ErrMaxValueExceeded = errors.New("maximum n value exceeded")
)

// CalculatorService handles the core logic for calculating Fibonacci numbers.
// It centralizes validation, algorithm retrieval, and execution options.
type CalculatorService struct {
	factory fibonacci.CalculatorFactory
	config  config.AppConfig
	maxN    uint64
}

// NewCalculatorService creates a new instance of CalculatorService.
//
// Parameters:
//   - factory: The factory to retrieve calculators from.
//   - cfg: The application configuration.
//   - maxN: The maximum allowed value for n (0 for no limit).
func NewCalculatorService(factory fibonacci.CalculatorFactory, cfg config.AppConfig, maxN uint64) *CalculatorService {
	return &CalculatorService{
		factory: factory,
		config:  cfg,
		maxN:    maxN,
	}
}

// Calculate retrieves the requested calculator and executes the calculation
// with the configured options. It also performs validation on the input n.
//
// Parameters:
//   - ctx: The context for cancellation.
//   - algoName: The name of the algorithm to use.
//   - n: The Fibonacci index to calculate.
//
// Returns:
//   - *big.Int: The result.
//   - error: An error if validation or calculation fails.
func (s *CalculatorService) Calculate(ctx context.Context, algoName string, n uint64) (*big.Int, error) {
	// Validation
	if s.maxN > 0 && n > s.maxN {
		return nil, ErrMaxValueExceeded
	}

	// Retrieve Algorithm
	calc, err := s.factory.Get(algoName)
	if err != nil {
		return nil, err
	}

	// Calculate with centralized options
	// Note: We pass nil for progressChan as this is intended for synchronous/service usage
	// where progress updates might not be needed or handled differently.
	return calc.Calculate(ctx, nil, 0, n, s.config.ToCalculationOptions())
}

