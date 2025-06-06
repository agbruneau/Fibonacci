package main

import (
	"context"
	"testing"
)

func TestFibBinetLarge(t *testing.T) {
	pool := newIntPool()
	n := 1000
	resFast, err := fibFastDoubling(context.Background(), nil, n, pool)
	if err != nil {
		t.Fatalf("fastdoubling error: %v", err)
	}
	resBinet, err := fibBinet(context.Background(), nil, n, pool)
	if err != nil {
		t.Fatalf("binet error: %v", err)
	}
	if resBinet.Cmp(resFast) != 0 {
		t.Errorf("Binet wrong for %d", n)
	}
}
