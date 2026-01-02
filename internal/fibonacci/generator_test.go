package fibonacci

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func TestIterativeGenerator_Next(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Expected first 15 Fibonacci numbers
	expected := []int64{0, 1, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377}

	for i, exp := range expected {
		val, err := gen.Next(ctx)
		if err != nil {
			t.Fatalf("Next() error at index %d: %v", i, err)
		}
		if val.Cmp(big.NewInt(exp)) != 0 {
			t.Errorf("F(%d) = %v, want %d", i, val, exp)
		}
	}
}

func TestIterativeGenerator_Current(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Before any Next(), Current() should return nil
	if gen.Current() != nil {
		t.Error("Current() should return nil before Next() is called")
	}

	// After first Next()
	val, _ := gen.Next(ctx)
	current := gen.Current()
	if current == nil {
		t.Fatal("Current() should not be nil after Next()")
	}
	if current.Cmp(val) != 0 {
		t.Errorf("Current() = %v, want %v", current, val)
	}

	// Verify Current() returns a copy (modifying it shouldn't affect generator)
	current.SetInt64(999)
	if gen.Current().Cmp(big.NewInt(0)) != 0 {
		t.Error("Current() should return a copy, not the internal value")
	}
}

func TestIterativeGenerator_Index(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Before Next(), index should be 0
	if gen.Index() != 0 {
		t.Errorf("Index() = %d, want 0 before Next()", gen.Index())
	}

	// After calling Next() multiple times
	for i := uint64(0); i < 10; i++ {
		_, _ = gen.Next(ctx)
		if gen.Index() != i {
			t.Errorf("After %d calls, Index() = %d, want %d", i+1, gen.Index(), i)
		}
	}
}

func TestIterativeGenerator_Reset(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Advance the generator
	for i := 0; i < 20; i++ {
		_, _ = gen.Next(ctx)
	}

	// Reset
	gen.Reset()

	// Check state is reset
	if gen.Index() != 0 {
		t.Errorf("After Reset(), Index() = %d, want 0", gen.Index())
	}
	if gen.Current() != nil {
		t.Error("After Reset(), Current() should return nil")
	}

	// Next() should return F(0)
	val, err := gen.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error after Reset(): %v", err)
	}
	if val.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("After Reset(), first Next() = %v, want 0", val)
	}
}

func TestIterativeGenerator_Skip_SmallJump(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Skip to F(10) = 55
	val, err := gen.Skip(ctx, 10)
	if err != nil {
		t.Fatalf("Skip(10) error: %v", err)
	}
	if val.Cmp(big.NewInt(55)) != 0 {
		t.Errorf("Skip(10) = %v, want 55", val)
	}

	// Index should be 10
	if gen.Index() != 10 {
		t.Errorf("After Skip(10), Index() = %d, want 10", gen.Index())
	}

	// Next() should return F(11) = 89
	next, _ := gen.Next(ctx)
	if next.Cmp(big.NewInt(89)) != 0 {
		t.Errorf("After Skip(10), Next() = %v, want 89", next)
	}
}

func TestIterativeGenerator_Skip_ToZero(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Advance first
	for i := 0; i < 5; i++ {
		_, _ = gen.Next(ctx)
	}

	// Skip to 0
	val, err := gen.Skip(ctx, 0)
	if err != nil {
		t.Fatalf("Skip(0) error: %v", err)
	}
	if val.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Skip(0) = %v, want 0", val)
	}
	if gen.Index() != 0 {
		t.Errorf("After Skip(0), Index() = %d, want 0", gen.Index())
	}
}

func TestIterativeGenerator_Skip_LargeJump(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Skip to F(1000) - this should use Calculator
	val, err := gen.Skip(ctx, 1000)
	if err != nil {
		t.Fatalf("Skip(1000) error: %v", err)
	}

	// Verify using a separate Calculator
	calc, _ := GlobalFactory().Get("fast")
	expected, _ := calc.Calculate(ctx, nil, 0, 1000, Options{})

	if val.Cmp(expected) != 0 {
		t.Errorf("Skip(1000) doesn't match Calculator result")
	}

	// Verify we can continue from here
	next, _ := gen.Next(ctx)
	expectedNext, _ := calc.Calculate(ctx, nil, 0, 1001, Options{})
	if next.Cmp(expectedNext) != 0 {
		t.Errorf("After Skip(1000), Next() doesn't match F(1001)")
	}
}

func TestIterativeGenerator_ContextCancellation(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Next() should return context error
	_, err := gen.Next(ctx)
	if err != context.Canceled {
		t.Errorf("Next() with cancelled context: got %v, want context.Canceled", err)
	}

	// Skip() should also return context error
	_, err = gen.Skip(ctx, 100)
	if err != context.Canceled {
		t.Errorf("Skip() with cancelled context: got %v, want context.Canceled", err)
	}
}

func TestIterativeGenerator_CopySemantics(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Get F(5) = 5
	for i := 0; i <= 5; i++ {
		_, _ = gen.Next(ctx)
	}

	// Verify returned value is a copy
	val1, _ := gen.Next(ctx) // F(6) = 8
	val1.SetInt64(0)         // Modify it

	val2 := gen.Current() // Should still be 8
	if val2.Cmp(big.NewInt(8)) != 0 {
		t.Errorf("Modifying returned value affected generator state")
	}
}

func TestIterativeGenerator_PropertyCassini(t *testing.T) {
	t.Parallel()

	// Test Cassini's identity: F(n-1)*F(n+1) - F(n)² = (-1)^n
	gen := NewIterativeGenerator()
	ctx := context.Background()

	// Generate first 50 Fibonacci numbers and verify Cassini's identity
	fibs := make([]*big.Int, 51)
	for i := 0; i <= 50; i++ {
		val, _ := gen.Next(ctx)
		fibs[i] = val
	}

	for n := 1; n < 50; n++ {
		// F(n-1) * F(n+1)
		product := new(big.Int).Mul(fibs[n-1], fibs[n+1])
		// F(n)²
		square := new(big.Int).Mul(fibs[n], fibs[n])
		// Difference
		diff := new(big.Int).Sub(product, square)

		// Should be (-1)^n
		expected := int64(-1)
		if n%2 == 0 {
			expected = 1
		}

		if diff.Cmp(big.NewInt(expected)) != 0 {
			t.Errorf("Cassini's identity failed for n=%d: F(%d)*F(%d) - F(%d)² = %v, want %d",
				n, n-1, n+1, n, diff, expected)
		}
	}
}

func TestIterativeGeneratorWithCalculator(t *testing.T) {
	t.Parallel()

	calc, _ := GlobalFactory().Get("matrix")
	gen := NewIterativeGeneratorWithCalculator(calc)
	ctx := context.Background()

	// Skip to a large number using the provided calculator
	val, err := gen.Skip(ctx, 500)
	if err != nil {
		t.Fatalf("Skip(500) error: %v", err)
	}

	// Verify using the same calculator
	expected, _ := calc.Calculate(ctx, nil, 0, 500, Options{})
	if val.Cmp(expected) != 0 {
		t.Errorf("Skip(500) with custom calculator doesn't match")
	}
}

func BenchmarkIterativeGenerator_Next(b *testing.B) {
	gen := NewIterativeGenerator()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Next(ctx)
	}
}

func BenchmarkIterativeGenerator_First1000(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := NewIterativeGenerator()
		for j := 0; j < 1000; j++ {
			_, _ = gen.Next(ctx)
		}
	}
}

func BenchmarkIterativeGenerator_Skip(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := NewIterativeGenerator()
		_, _ = gen.Skip(ctx, 10000)
	}
}

func TestIterativeGenerator_Timeout(t *testing.T) {
	t.Parallel()

	gen := NewIterativeGenerator()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Let the timeout expire
	time.Sleep(5 * time.Millisecond)

	_, err := gen.Next(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Next() with expired timeout: got %v, want context.DeadlineExceeded", err)
	}
}
