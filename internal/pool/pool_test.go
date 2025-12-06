package pool

import (
	"math/big"
	"testing"
)

func TestAcquireBigInt(t *testing.T) {
	// Acquire a big.Int
	z := AcquireBigInt()
	if z == nil {
		t.Fatal("AcquireBigInt returned nil")
	}

	// Verify it is initialized to 0
	if z.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("AcquireBigInt returned %v, expected 0", z)
	}

	// modify it
	z.SetInt64(123)

	// Release it
	ReleaseBigInt(z)

	// Acquire another one, it should be 0 (either reused and reset, or new)
	z2 := AcquireBigInt()
	if z2.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("AcquireBigInt (2nd call) returned %v, expected 0", z2)
	}
}

func TestReleaseBigInt_Nil(t *testing.T) {
	// Should not panic
	ReleaseBigInt(nil)
}
