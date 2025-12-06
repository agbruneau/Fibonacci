package bigfft

import (
	"testing"
)

func TestPreWarmPools(t *testing.T) {
	// Test PreWarmPools does not panic and allocates pools
	// Pass uint64 directly as per signature
	PreWarmPools(1000)

	// Try with larger n
	PreWarmPools(100000)
}
