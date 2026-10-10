// Package clock times short intervals at sub-microsecond resolution on every
// platform.
//
// On Windows, Go's runtime clock — time.Now, monotonic and wall readings alike
// — advances in steps of about 0.5 ms: an interval shorter than that reads as
// 0 and a longer one is rounded to a multiple of the step. The testing package
// works around this the same way, with QueryPerformanceCounter
// (src/testing/testing_windows.go). Everywhere else the runtime clock is
// already fine-grained and this package is a thin wrapper over it.
//
// Use it to measure what is shown or compared: calculation durations and
// calibration timings. A deadline, a timeout or a timestamp stays on time.
package clock

import (
	"math/bits"
	"time"
)

// Stamp is a reading of the clock; it only means something to Since.
type Stamp struct{ ticks int64 }

// Now reads the clock.
func Now() Stamp { return Stamp{ticks()} }

// Since returns the time elapsed since s.
func Since(s Stamp) time.Duration {
	delta := ticks() - s.ticks
	if delta <= 0 {
		return 0
	}
	// delta·10⁹ overflows int64 after ~15 min at a 10 MHz counter; the 128-bit
	// product does not, and the quotient fits for any interval under 292 years.
	hi, lo := bits.Mul64(uint64(delta), uint64(time.Second))
	quo, _ := bits.Div64(hi, lo, uint64(frequency))
	return time.Duration(quo)
}
