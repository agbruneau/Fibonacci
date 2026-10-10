//go:build !windows

package clock

import "time"

const frequency = int64(time.Second)

var origin = time.Now()

// ticks counts nanoseconds on the runtime's monotonic clock.
func ticks() int64 { return int64(time.Since(origin)) }
