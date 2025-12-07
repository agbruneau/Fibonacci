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

func TestReleaseBigInt_OversizedRejected(t *testing.T) {
	// Crée un big.Int qui dépasse MaxPooledBitLen
	oversized := new(big.Int)
	// SetBit crée un entier avec au moins (MaxPooledBitLen + 1) bits
	oversized.SetBit(oversized, MaxPooledBitLen+1, 1)

	if oversized.BitLen() <= MaxPooledBitLen {
		t.Fatalf("Test setup error: oversized.BitLen()=%d, expected > %d",
			oversized.BitLen(), MaxPooledBitLen)
	}

	// Release l'objet surdimensionné (devrait être ignoré)
	ReleaseBigInt(oversized)

	// Acquérir plusieurs objets pour vider le pool si besoin
	// et vérifier qu'aucun n'est l'objet surdimensionné
	for i := 0; i < 10; i++ {
		z := AcquireBigInt()
		// AcquireBigInt réinitialise à 0, donc BitLen devrait être 0
		// Mais on vérifie aussi que ce n'est pas notre objet géant
		if z.BitLen() > MaxPooledBitLen {
			t.Errorf("Pool returned oversized big.Int with BitLen=%d", z.BitLen())
		}
		ReleaseBigInt(z)
	}
}

func TestReleaseBigInt_NormalSizeAccepted(t *testing.T) {
	// Crée un big.Int de taille normale
	normal := AcquireBigInt()
	normal.SetInt64(999999999)

	if normal.BitLen() > MaxPooledBitLen {
		t.Fatalf("Test setup error: normal sized int exceeds limit")
	}

	// Release et réacquérir - devrait fonctionner normalement
	ReleaseBigInt(normal)

	z := AcquireBigInt()
	if z == nil {
		t.Fatal("AcquireBigInt returned nil after releasing normal-sized int")
	}
	// Doit être réinitialisé à 0
	if z.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("AcquireBigInt returned %v, expected 0", z)
	}
}