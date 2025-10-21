//go:build linux

package main

import (
	"syscall"
	"time"
)

// getCPUTime returns the CPU time used by the calling thread.
func getCPUTime() (time.Duration, error) {
	var rusage syscall.Rusage
	// On Linux, getrusage can be called with RUSAGE_THREAD to get stats for the current thread.
	// This is not defined in the standard syscall package, so we use the value 1.
	const RUSAGE_THREAD = 1
	err := syscall.Getrusage(RUSAGE_THREAD, &rusage)
	if err != nil {
		return 0, err
	}
	return time.Duration(rusage.Utime.Sec*1e9 + rusage.Utime.Usec*1e3), nil
}
