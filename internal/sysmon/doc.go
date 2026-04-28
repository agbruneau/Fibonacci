// Package sysmon provides system-wide CPU and memory usage sampling.
//
// # Role
//
// sysmon offers a single, dependency-light entry point — Sample — that
// returns a snapshot of host-level resource usage. It is consumed by the
// metrics and TUI packages to surface live CPU/memory pressure alongside
// per-calculation indicators.
//
// # Invariants
//
//   - Sample never returns an error: probe failures yield zero values
//     so that callers can render a degraded but still-non-nil snapshot.
//   - CPU sampling uses interval=0 (delta since last call); the first
//     call after process start may report 0% until a baseline is built.
//
// # Example
//
//	s := sysmon.Sample()
//	fmt.Printf("CPU %.1f%%  Mem %.1f%%\n", s.CPUPercent, s.MemPercent)
package sysmon
