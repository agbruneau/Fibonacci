package tui

import (
	"testing"
)

func TestRingBuffer_PushAndSlice(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	got := rb.Slice()
	want := []float64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %f, want %f", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1

	got := rb.Slice()
	want := []float64{2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %f, want %f", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Last(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(5)
	if rb.Last() != 0 {
		t.Error("expected 0 for empty buffer")
	}
	rb.Push(10)
	rb.Push(20)
	rb.Push(30)
	if rb.Last() != 30 {
		t.Errorf("expected 30, got %f", rb.Last())
	}
}

func TestRingBuffer_Last_AfterOverflow(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(2)
	rb.Push(10)
	rb.Push(20)
	rb.Push(30) // overwrites 10
	if rb.Last() != 30 {
		t.Errorf("expected 30, got %f", rb.Last())
	}
}

func TestRingBuffer_Reset(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(5)
	rb.Push(1)
	rb.Push(2)
	rb.Reset()

	if rb.Len() != 0 {
		t.Errorf("expected len 0, got %d", rb.Len())
	}
	if rb.Slice() != nil {
		t.Error("expected nil slice after reset")
	}
}

func TestRingBuffer_Resize_Grow(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Resize(5)

	if rb.Cap() != 5 {
		t.Errorf("expected cap 5, got %d", rb.Cap())
	}
	got := rb.Slice()
	want := []float64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %f, want %f", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Resize_Shrink(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(5)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4)
	rb.Push(5)
	rb.Resize(3) // keep most recent: 3, 4, 5

	got := rb.Slice()
	want := []float64{3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %f, want %f", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_ZeroCapacity(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(0)
	if rb.Cap() != 1 {
		t.Errorf("expected min cap 1, got %d", rb.Cap())
	}
	rb.Push(42)
	if rb.Last() != 42 {
		t.Errorf("expected 42, got %f", rb.Last())
	}
}

func TestRingBuffer_Resize_SameCapacity(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(3)
	rb.Push(1)
	rb.Push(2)
	rb.Resize(3) // no-op

	if rb.Len() != 2 {
		t.Errorf("expected len 2 after same-cap resize, got %d", rb.Len())
	}
}

func TestRenderSparkline_Empty(t *testing.T) {
	t.Parallel()
	got := RenderSparkline(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRenderSparkline_AllZero(t *testing.T) {
	t.Parallel()
	got := RenderSparkline([]float64{0, 0, 0})
	runes := []rune(got)
	for i, r := range runes {
		if r != '▁' {
			t.Errorf("index %d: expected '▁', got %c", i, r)
		}
	}
}

func TestRenderSparkline_AllMax(t *testing.T) {
	t.Parallel()
	got := RenderSparkline([]float64{100, 100, 100})
	runes := []rune(got)
	for i, r := range runes {
		if r != '█' {
			t.Errorf("index %d: expected '█', got %c", i, r)
		}
	}
}

func TestRenderSparkline_Gradient(t *testing.T) {
	t.Parallel()
	values := []float64{0, 14.3, 28.6, 42.9, 57.1, 71.4, 85.7, 100}
	got := RenderSparkline(values)
	runes := []rune(got)
	if len(runes) != 8 {
		t.Fatalf("expected 8 chars, got %d", len(runes))
	}
	// Should be strictly ascending
	for i := 1; i < len(runes); i++ {
		if runes[i] < runes[i-1] {
			t.Errorf("expected ascending at index %d: %c < %c", i, runes[i], runes[i-1])
		}
	}
}

func TestRenderSparkline_Clamping(t *testing.T) {
	t.Parallel()
	got := RenderSparkline([]float64{-10, 150})
	runes := []rune(got)
	if runes[0] != '▁' {
		t.Errorf("negative not clamped to min: got %c", runes[0])
	}
	if runes[1] != '█' {
		t.Errorf("over-100 not clamped to max: got %c", runes[1])
	}
}

func TestRenderSparkline_MidValue(t *testing.T) {
	t.Parallel()
	got := RenderSparkline([]float64{50})
	runes := []rune(got)
	// 50/100 * 7 = 3.5 -> index 3 -> '▄'
	if runes[0] != '▄' {
		t.Errorf("expected '▄' for 50%%, got %c", runes[0])
	}
}
