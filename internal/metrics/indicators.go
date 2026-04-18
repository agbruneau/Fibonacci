package metrics

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"time"
)

// Indicators holds metrics that provide insight into both the performance
// characteristics and mathematical properties of a Fibonacci computation.
// During execution, performance indicators are estimated from progress;
// after completion, all indicators (including mathematical ones) are exact.
type Indicators struct {
	// Performance
	BitsPerSecond   float64 // bits of result produced per second
	DigitsPerSecond float64 // estimated decimal digits produced per second
	DoublingSteps   uint64  // number of doubling iterations ≈ log₂(n)
	StepsPerSecond  float64 // doubling steps executed per second

	// Mathematical (only available after calculation completes)
	GoldenRatioDeviation float64 // % deviation of actual bitLen vs theoretical n·log₂(φ)
	DigitalRoot          int     // iterative digit sum until single digit (1-9)
	LastDigits           string  // last 20 decimal digits of F(n)
	IsEven               bool    // true when F(n) is even (iff 3 | n)

	// Live is true when indicators are estimated from progress, false when final.
	Live bool
}

// log2Phi is log₂(φ) where φ = (1+√5)/2 ≈ 1.6180339887.
// Used for theoretical bit-length estimation: F(n) ≈ φⁿ/√5 → bitLen ≈ n·log₂(φ).
var log2Phi = math.Log2(math.Phi)

// lastDigitsMod is 10^20, used to extract the last 20 decimal digits via modular arithmetic.
var lastDigitsMod = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)

// ComputeLive estimates indicators from progress data during execution.
// Performance metrics are derived from the theoretical result size (n·log₂(φ))
// scaled by progress. Mathematical indicators that require the actual result
// (digital root, last digits, golden ratio deviation) are left at zero.
func ComputeLive(n uint64, progress float64, elapsed time.Duration) *Indicators {
	if elapsed <= 0 || progress <= 0 || n <= 1 {
		return &Indicators{
			// bits.Len64 returns int in [0, 64], safe to cast to uint64. #nosec G115
			DoublingSteps: uint64(bits.Len64(n)),
			IsEven:        n%3 == 0,
			Live:          true,
		}
	}

	seconds := elapsed.Seconds()
	theoreticalBits := float64(n) * log2Phi
	estimatedBitsProduced := progress * theoreticalBits
	estimatedDigitsProduced := estimatedBitsProduced * math.Log10(2)
	// bits.Len64 returns int in [0, 64], safe to cast to uint64. #nosec G115
	doublingSteps := uint64(bits.Len64(n))
	completedSteps := progress * float64(doublingSteps)

	return &Indicators{
		BitsPerSecond:   estimatedBitsProduced / seconds,
		DigitsPerSecond: estimatedDigitsProduced / seconds,
		DoublingSteps:   doublingSteps,
		StepsPerSecond:  completedSteps / seconds,
		IsEven:          n%3 == 0,
		Live:            true,
	}
}

// Compute calculates all indicators from a completed Fibonacci result.
// It performs only O(1) or cheap big.Int operations (Mod, BitLen) and
// avoids full-result string conversion to keep overhead minimal.
func Compute(result *big.Int, n uint64, duration time.Duration) *Indicators {
	if result == nil || duration <= 0 {
		return &Indicators{}
	}

	bitLen := result.BitLen()
	seconds := duration.Seconds()
	estimatedDigits := float64(bitLen) * math.Log10(2)
	// bits.Len64 returns int in [0, 64], safe to cast to uint64. #nosec G115
	doublingSteps := uint64(bits.Len64(n))

	ind := &Indicators{
		BitsPerSecond:   float64(bitLen) / seconds,
		DigitsPerSecond: estimatedDigits / seconds,
		DoublingSteps:   doublingSteps,
		StepsPerSecond:  float64(doublingSteps) / seconds,
		IsEven:          n%3 == 0,
	}

	// Golden ratio deviation: compare actual bitLen to theoretical n·log₂(φ)
	if n > 1 {
		theoretical := float64(n) * log2Phi
		ind.GoldenRatioDeviation = math.Abs(float64(bitLen)-theoretical) / theoretical * 100
	}

	// Digital root: 1 + ((x - 1) mod 9) for x > 0
	ind.DigitalRoot = digitalRoot(result)

	// Last 20 digits via modular arithmetic (no full string conversion)
	ind.LastDigits = lastNDigits(result, 20)

	return ind
}

// digitalRoot computes the digital root of x (repeated digit sum until single digit).
// For positive integers: digitalRoot(x) = 1 + ((x - 1) mod 9).
// Uses big.Int.Mod which is efficient on arbitrary-precision integers.
func digitalRoot(x *big.Int) int {
	if x.Sign() <= 0 {
		return 0
	}

	nine := big.NewInt(9)
	// (x - 1) mod 9
	tmp := new(big.Int).Sub(x, big.NewInt(1))
	tmp.Mod(tmp, nine)
	return int(tmp.Int64()) + 1
}

// lastNDigits returns the last n decimal digits of x as a zero-padded string.
// Uses big.Int.Mod(x, 10^n) to avoid converting the entire number to a string.
func lastNDigits(x *big.Int, n int) string {
	if x.Sign() <= 0 {
		return "0"
	}

	mod := lastDigitsMod
	if n != 20 {
		mod = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	}

	last := new(big.Int).Mod(x, mod)
	s := last.String()

	// Zero-pad if the result has fewer than n digits (leading zeros)
	if len(s) < n && x.BitLen() > n*4 { // only pad if x is large enough to have n digits
		s = fmt.Sprintf("%0*s", n, s)
	}
	return s
}

// FormatBitsPerSecond formats a bits/s value with appropriate unit suffix.
func FormatBitsPerSecond(bps float64) string {
	switch {
	case bps >= 1e9:
		return fmt.Sprintf("%.2f Gbit/s", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f Mbit/s", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f Kbit/s", bps/1e3)
	default:
		return fmt.Sprintf("%.0f bit/s", bps)
	}
}

// FormatDigitsPerSecond formats a digits/s value with appropriate unit suffix.
func FormatDigitsPerSecond(dps float64) string {
	switch {
	case dps >= 1e6:
		return fmt.Sprintf("%.2f M digits/s", dps/1e6)
	case dps >= 1e3:
		return fmt.Sprintf("%.2f K digits/s", dps/1e3)
	default:
		return fmt.Sprintf("%.0f digits/s", dps)
	}
}
