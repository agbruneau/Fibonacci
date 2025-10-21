//go:build !linux

package main

import "time"

// getCPUTime returns a zero value for unsupported platforms.
func getCPUTime() (time.Duration, error) {
	return 0, nil
}
