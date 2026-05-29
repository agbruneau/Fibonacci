package tui

// sparklineChars maps values 0..7 to Unicode block elements ▁▂▃▄▅▆▇█.
var sparklineChars = [8]rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// The RingBuffer type used to live here. It has been generalised and moved to
// ringbuffer.go so that LogsModel can share the same fixed-capacity circular
// storage (audit task R4.8). NewRingBuffer remains a float64 constructor for
// backward compatibility with sparkline call sites.

// RenderSparkline converts values (0..100) into a sparkline string using Unicode blocks.
func RenderSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	runes := make([]rune, len(values))
	for i, v := range values {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		idx := int(v / 100.0 * 7.0)
		if idx > 7 {
			idx = 7
		}
		runes[i] = sparklineChars[idx]
	}
	return string(runes)
}

// brailleDots maps (col 0-1, row 0-3) to the braille dot bit offsets.
// Braille character = U+2800 + sum of activated dot bits.
// Column 0: dots 1,2,3,7 (bits 0,1,2,6)
// Column 1: dots 4,5,6,8 (bits 3,4,5,7)
var brailleDots = [2][4]rune{
	{0x01, 0x02, 0x04, 0x40}, // left column
	{0x08, 0x10, 0x20, 0x80}, // right column
}

// RenderBrailleChart renders values (0..100) as a multi-row braille dot chart.
// Each braille character is 2 columns wide × 4 rows tall in the dot grid.
// The chart has `rows` text rows and `width` character columns.
// values are plotted right-aligned (most recent on the right).
func RenderBrailleChart(values []float64, width, rows int) []string {
	if width <= 0 || rows <= 0 || len(values) == 0 {
		return nil
	}

	// Total dot rows available in the braille grid
	dotRows := rows * 4
	// Total dot columns: each character covers 2 dot columns
	dotCols := width * 2

	// Initialize the braille grid (rows × width characters, all empty U+2800)
	grid := make([][]rune, rows)
	for r := range grid {
		grid[r] = make([]rune, width)
		for c := range grid[r] {
			grid[r][c] = 0x2800
		}
	}

	// Plot each value as a dot in the grid (right-aligned)
	startIdx := 0
	if len(values) > dotCols {
		startIdx = len(values) - dotCols
	}

	for i := startIdx; i < len(values); i++ {
		dotCol := (i - startIdx) + (dotCols - min(len(values), dotCols))
		plotBrailleValue(grid, width, rows, dotCol, values[i], dotRows)
	}

	// Convert grid to strings
	result := make([]string, rows)
	for r := range grid {
		result[r] = string(grid[r])
	}
	return result
}

// plotBrailleValue clamps v into [0,100], maps it to a dot row, and activates
// the corresponding braille dot in grid at the given dot column. grid is mutated
// in place; out-of-range cells are ignored.
func plotBrailleValue(grid [][]rune, width, rows, dotCol int, v float64, dotRows int) {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}

	// Map value to dot row (0 = top, dotRows-1 = bottom)
	dotRow := dotRows - 1 - int(v/100.0*float64(dotRows-1))
	if dotRow < 0 {
		dotRow = 0
	}
	if dotRow >= dotRows {
		dotRow = dotRows - 1
	}

	// Convert dot coordinates to character cell + offset within cell
	charCol := dotCol / 2
	charRow := dotRow / 4
	subCol := dotCol % 2
	subRow := dotRow % 4

	if charCol >= 0 && charCol < width && charRow >= 0 && charRow < rows {
		grid[charRow][charCol] |= brailleDots[subCol][subRow]
	}
}
