// Package system provides system-wide CPU and memory usage sampling.
package system

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// Stats holds a single snapshot of system-wide resource usage.
type Stats struct {
	CPUPercent float64 // 0.0 .. 100.0
	MemPercent float64 // 0.0 .. 100.0
}

// Sample collects a single system-wide CPU and memory snapshot.
// CPU uses interval=0 (delta since last call). Returns zero values on error.
func Sample() Stats {
	var s Stats
	cpuPcts, err := cpu.Percent(0, false)
	if err == nil && len(cpuPcts) > 0 {
		s.CPUPercent = cpuPcts[0]
	}
	vmem, err := mem.VirtualMemory()
	if err == nil && vmem != nil {
		s.MemPercent = vmem.UsedPercent
	}
	return s
}
