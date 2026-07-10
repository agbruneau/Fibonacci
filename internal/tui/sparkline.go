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
		v = min(max(v, 0), 100)
		idx := min(int(v/100.0*7.0), 7)
		runes[i] = sparklineChars[idx]
	}
	return string(runes)
}
