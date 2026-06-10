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

func TestRingBuffer_Resize_NonPositiveCap(t *testing.T) {
	t.Parallel()
	for _, newCap := range []int{0, -3} {
		rb := NewRingBuffer(3)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)
		rb.Resize(newCap)

		if rb.Cap() != 1 {
			t.Errorf("Resize(%d): cap = %d, want 1", newCap, rb.Cap())
		}
		// The most recent sample must survive the forced shrink to 1.
		if rb.Len() != 1 {
			t.Errorf("Resize(%d): len = %d, want 1", newCap, rb.Len())
		}
		if rb.Last() != 3 {
			t.Errorf("Resize(%d): last = %f, want 3", newCap, rb.Last())
		}
	}
}

func TestRenderBrailleChart_EmptyAndInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		values []float64
		width  int
		rows   int
	}{
		{"nil values", nil, 5, 2},
		{"empty values", []float64{}, 3, 3},
		{"zero width", []float64{1}, 0, 2},
		{"negative width", []float64{1}, -1, 3},
		{"zero rows", []float64{1}, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RenderBrailleChart(tt.values, tt.width, tt.rows); got != nil {
				t.Errorf("expected nil, got %q", got)
			}
		})
	}
}

// TestRenderBrailleChart_SingleCellDots pins the exact braille rune produced
// for one value in a 1x1 chart: the dot must land in the right dot column
// (right-aligned plotting) at the row matching the value.
func TestRenderBrailleChart_SingleCellDots(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		// dot bits from brailleDots: right column rows 0..3 = 0x08,0x10,0x20,0x80
		{"zero plots bottom right", 0, "⢀"},
		{"max plots top right", 100, "⠈"},
		{"mid plots third row", 50, "⠠"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RenderBrailleChart([]float64{tt.value}, 1, 1)
			if len(got) != 1 {
				t.Fatalf("expected 1 row, got %d", len(got))
			}
			if got[0] != tt.want {
				t.Errorf("got %q (%x), want %q (%x)", got[0], got[0], tt.want, tt.want)
			}
		})
	}
}

func TestRenderBrailleChart_ClampsOutOfRangeValues(t *testing.T) {
	t.Parallel()
	// -50 must clamp to the bottom-left dot (0x40), 150 to the top-right dot
	// (0x08); both share the single cell of a 1x1 chart.
	got := RenderBrailleChart([]float64{-50, 150}, 1, 1)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	if want := "⡈"; got[0] != want {
		t.Errorf("got %q (%x), want %q (%x)", got[0], got[0], want, want)
	}
}

func TestRenderBrailleChart_TruncatesToWidth(t *testing.T) {
	t.Parallel()
	// Four values but only two dot columns: the two old zeros must be dropped,
	// leaving only the two recent max values as top dots (0x01 | 0x08).
	got := RenderBrailleChart([]float64{0, 0, 100, 100}, 1, 1)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	if want := "⠉"; got[0] != want {
		t.Errorf("got %q (%x), want %q (%x): stale values were not truncated", got[0], got[0], want, want)
	}
}

func TestRenderBrailleChart_MultiRowBottomPlacement(t *testing.T) {
	t.Parallel()
	// With two text rows the dot grid is 8 rows tall: a zero value must land
	// in the bottom text row, leaving the top row empty.
	got := RenderBrailleChart([]float64{0}, 1, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
	if got[0] != "⠀" {
		t.Errorf("top row = %q, want empty braille cell", got[0])
	}
	if got[1] != "⢀" {
		t.Errorf("bottom row = %q (%x), want ⢀", got[1], got[1])
	}
}

func TestRenderBrailleChart_GridProperties(t *testing.T) {
	t.Parallel()
	values := make([]float64, 40)
	for i := range values {
		values[i] = float64(i*100) / 39
	}

	const width, rows = 10, 3
	got := RenderBrailleChart(values, width, rows)
	if len(got) != rows {
		t.Fatalf("expected %d rows, got %d", rows, len(got))
	}
	plotted := false
	for r, row := range got {
		runes := []rune(row)
		if len(runes) != width {
			t.Errorf("row %d: expected %d runes, got %d", r, width, len(runes))
		}
		for c, ch := range runes {
			if ch < 0x2800 || ch > 0x28FF {
				t.Errorf("row %d col %d: U+%04X outside braille block", r, c, ch)
			}
			if ch != 0x2800 {
				plotted = true
			}
		}
	}
	if !plotted {
		t.Error("expected at least one plotted dot in the grid")
	}
}

// TestPlotBrailleValue_IgnoresOutOfGridCells protects the documented
// contract that out-of-range cells are ignored: the call must neither panic
// nor mutate any cell.
func TestPlotBrailleValue_IgnoresOutOfGridCells(t *testing.T) {
	t.Parallel()
	newGrid := func() [][]rune {
		return [][]rune{{0x2800, 0x2800}}
	}
	assertUntouched := func(t *testing.T, grid [][]rune) {
		t.Helper()
		for r, row := range grid {
			for c, ch := range row {
				if ch != 0x2800 {
					t.Errorf("grid[%d][%d] = U+%04X, want untouched U+2800", r, c, ch)
				}
			}
		}
	}

	t.Run("dot column beyond grid", func(t *testing.T) {
		t.Parallel()
		grid := newGrid()
		// dotCol 4 maps to charCol 2 >= width: the write must be skipped.
		plotBrailleValue(grid, 2, 1, 4, 50, 4)
		assertUntouched(t, grid)
	})

	t.Run("degenerate zero dotRows", func(t *testing.T) {
		t.Parallel()
		grid := newGrid()
		// dotRows=0 drives dotRow through both defensive clamps; the negative
		// dot column keeps the resulting cell out of range, so the call must
		// be a strict no-op instead of a panic or an out-of-bounds write.
		plotBrailleValue(grid, 2, 1, -2, 50, 0)
		assertUntouched(t, grid)
	})
}
