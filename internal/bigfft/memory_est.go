package bigfft

import "math/bits"

// MemoryEstimate contains estimated maximum sizes for various data structures
// needed during the calculation of F(n).
type MemoryEstimate struct {
	MaxWordSliceSize   int
	MaxFermatSize      int
	MaxNatSliceSize    int
	MaxFermatSliceSize int
}

// EstimateMemoryNeeds estimates the memory requirements for calculating F(n).
// This is a heuristic estimation used for pool pre-warming.
func EstimateMemoryNeeds(n uint64) MemoryEstimate {
	// F(n) has approximately n * log10(phi) / log10(2) bits
	// log2(phi) ≈ 0.69424
	bitLen := uint64(float64(n) * 0.69424)
	wordLen := int((bitLen + 63) / 64)

	// In the worst case (FFT multiplication), we need buffers larger than the number itself
	// The FFT size K is roughly related to the number of words

	// Estimate K (FFT size)
	// K is a power of 2 such that K * m * 64 > 2 * wordLen * 64
	// Rough approximation for pre-warming

	// Max word slice: large enough to hold the number + overhead
	maxWordSlice := wordLen * 2

	// Max fermat size depends on K and m.
	// For very large N, fermat size can be significant.
	// We'll use a conservative estimate based on word length.
	maxFermat := 2048 // Default reasonable max for pool warming
	if wordLen > 100000 {
		maxFermat = 8192
	}

	// Max slice sizes (number of coefficients)
	// Typically K, which can be 1024, 2048, etc.
	maxNatSlice := 2048
	maxFermatSlice := 2048

	// Determine K roughly
	if wordLen > 0 {
		k := bits.Len(uint(wordLen))
		if k > 10 {
			kVal := 1 << (k - 3) // Heuristic
			if kVal > maxNatSlice {
				maxNatSlice = kVal
				maxFermatSlice = kVal
			}
		}
	}

	return MemoryEstimate{
		MaxWordSliceSize:   maxWordSlice,
		MaxFermatSize:      maxFermat,
		MaxNatSliceSize:    maxNatSlice,
		MaxFermatSliceSize: maxFermatSlice,
	}
}
