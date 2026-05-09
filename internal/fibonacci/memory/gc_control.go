package memory

import (
	"math"
	"runtime"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// GCMode controls the garbage collector behavior during calculation.
type GCMode string

const (
	GCModeAuto       GCMode = "auto"
	GCModeAggressive GCMode = "aggressive"
	GCModeDisabled   GCMode = "disabled"
)

// GCAutoThreshold is the minimum N for auto GC control to activate.
const GCAutoThreshold uint64 = 1_000_000

// DefaultMemoryLimitMultiplier is the factor applied to runtime.MemStats.Sys
// to derive the soft GOMEMLIMIT installed while GC is disabled. It acts as
// an OOM safety net: if the calculation runs away, the Go runtime will
// trigger emergency GC instead of letting the process consume unbounded
// memory.
//
// The value (3.0) was chosen empirically — see audit R4.2 — to leave room
// for the doubling-loop working set (a + b + temp + arena) without giving
// up the OOM guard. The same value is mirrored in
// config.DefaultThresholdTuning.MemoryLimitMultiplier; this package owns
// the canonical value because internal/config already imports
// internal/fibonacci/memory, so the reverse direction would be an import
// cycle.
const DefaultMemoryLimitMultiplier = 3.0

// GCController manages Go's garbage collector during intensive calculations.
// It disables GC during computation and restores it afterward, reducing
// pause times and memory overhead for large calculations.
type GCController struct {
	mode              GCMode
	originalGCPercent int
	active            bool
	logger            zerolog.Logger
	startStats        runtime.MemStats
	endStats          runtime.MemStats
}

// GCStats holds GC statistics for a calculation.
type GCStats struct {
	HeapAlloc    uint64
	TotalAlloc   uint64
	NumGC        uint32
	PauseTotalNs uint64
}

// NewGCController creates a GC controller for the given mode and N.
func NewGCController(mode string, n uint64) *GCController {
	gc := &GCController{mode: GCMode(mode), logger: zerolog.Nop()}
	switch gc.mode {
	case GCModeAggressive:
		gc.active = true
	case GCModeAuto:
		gc.active = n >= GCAutoThreshold
	default:
		gc.active = false
	}
	return gc
}

// SetLogger configures the logger for GC control events.
func (gc *GCController) SetLogger(l zerolog.Logger) {
	gc.logger = l
}

// WithGC executes fn under GC control: it disables GC via Begin before running
// fn and guarantees restoration via a deferred End, even if fn panics. If fn
// panics, GC settings are restored before the panic is re-raised, preventing
// the silent process-wide GC leak that occurs when Begin/End are used directly
// and a panic escapes between the two calls.
//
// Returns the error returned by fn (nil on success).
func (gc *GCController) WithGC(fn func() error) (err error) {
	gc.Begin()
	defer gc.End()
	return fn()
}

// Deprecated: Use WithGC() for panic-safe usage. Begin/End used directly leak
// GC settings if a panic escapes between the two calls.
//
// Begin disables GC if the controller is active.
func (gc *GCController) Begin() {
	if !gc.active {
		return
	}
	runtime.ReadMemStats(&gc.startStats)
	gc.originalGCPercent = debug.SetGCPercent(-1)
	// Set soft memory limit as OOM safety net.
	if gc.startStats.Sys > 0 {
		limit := int64(float64(gc.startStats.Sys) * DefaultMemoryLimitMultiplier)
		if limit > 0 {
			debug.SetMemoryLimit(limit)
		}
	}
	gc.logger.Debug().
		Str("mode", string(gc.mode)).
		Uint64("heap_alloc_bytes", gc.startStats.HeapAlloc).
		Msg("gc disabled")
}

// End restores original GC settings and triggers a collection.
func (gc *GCController) End() {
	if !gc.active {
		return
	}
	runtime.ReadMemStats(&gc.endStats)
	debug.SetGCPercent(gc.originalGCPercent)
	debug.SetMemoryLimit(math.MaxInt64)
	runtime.GC()
	gc.logger.Debug().
		Str("mode", string(gc.mode)).
		Uint64("heap_alloc_bytes", gc.endStats.HeapAlloc).
		Uint64("total_alloc_bytes", gc.endStats.TotalAlloc-gc.startStats.TotalAlloc).
		Uint32("gc_cycles", gc.endStats.NumGC-gc.startStats.NumGC).
		Msg("gc re-enabled")
}

// Stats returns GC statistics delta between Begin and End.
func (gc *GCController) Stats() GCStats {
	return GCStats{
		HeapAlloc:    gc.endStats.HeapAlloc,
		TotalAlloc:   gc.endStats.TotalAlloc - gc.startStats.TotalAlloc,
		NumGC:        gc.endStats.NumGC - gc.startStats.NumGC,
		PauseTotalNs: gc.endStats.PauseTotalNs - gc.startStats.PauseTotalNs,
	}
}
