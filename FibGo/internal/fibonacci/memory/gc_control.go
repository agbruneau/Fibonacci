package memory

import (
	"context"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
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
// The value (3.0) is a headroom setting, not a measured one: it is meant to
// leave room for the doubling-loop working set (a + b + temp + arena)
// without giving up the OOM guard. Audit R4.2 named the constant and moved
// it here; it did not measure it, and no artifact in the repo pins 3.0
// against any other factor.
const DefaultMemoryLimitMultiplier = 3.0

// Concurrency-safe process-wide GC control (A2-01 / ADR-0005).
//
// GOGC and the soft memory limit are process-global, but in comparison mode
// (--algo all, N >= GCAutoThreshold) several GCControllers run concurrently via
// errgroup. A naive per-controller save/restore is unsafe: the second Begin
// captures -1 as its "original", and the first End restores GOGC (or drops the
// OOM memory limit) while a sibling is still computing a huge number.
//
// We serialize through a package-level refcount: only the first active Begin
// disables GC and records the TRUE original GOGC and soft memory limit; only the
// last active End restores both. Restoring (not zeroing) the memory limit keeps
// a user-supplied GOMEMLIMIT in effect across calculations. This preserves the
// panic-safe WithGC contract while making concurrent activation correct.
var (
	gcGlobalMu      sync.Mutex
	gcActiveDepth   int
	gcSavedPercent  int
	gcSavedMemLimit int64
)

// GCController manages Go's garbage collector during intensive calculations.
// It disables GC during computation and restores it afterward, reducing
// pause times and memory overhead for large calculations.
type GCController struct {
	mode       GCMode
	active     bool
	logger     *slog.Logger
	startStats runtime.MemStats
	endStats   runtime.MemStats
}

// NewGCController creates a GC controller for the given mode and N.
//
// logger receives the "gc disabled" / "gc re-enabled" records, which report the
// heap size at each boundary and the number of cycles that ran anyway. A nil
// logger discards them.
//
// Before audit OBS-01 the logger was not a parameter at all: it defaulted to a
// no-op and the only way to change it was a test-only setter, so these two
// records — the ones that answer "did GOGC=off actually hold?" — could not be
// seen from the binary at any verbosity.
func NewGCController(mode string, n uint64, logger *slog.Logger) *GCController {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	gc := &GCController{mode: GCMode(mode), logger: logger}
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

	gcGlobalMu.Lock()
	if gcActiveDepth == 0 {
		// First active controller: capture the TRUE original GOGC and soft
		// memory limit, disable GC process-wide, then install our own soft
		// memory limit as an OOM net. SetMemoryLimit(-1) reads the current
		// limit without changing it.
		gcSavedPercent = debug.SetGCPercent(-1)
		gcSavedMemLimit = debug.SetMemoryLimit(-1)
		if gc.startStats.Sys > 0 {
			if limit := int64(float64(gc.startStats.Sys) * DefaultMemoryLimitMultiplier); limit > 0 {
				debug.SetMemoryLimit(limit)
			}
		}
	}
	gcActiveDepth++
	gcGlobalMu.Unlock()

	gc.logger.LogAttrs(context.Background(), slog.LevelDebug, "gc disabled",
		slog.String("mode", string(gc.mode)),
		slog.Uint64("heap_alloc_bytes", gc.startStats.HeapAlloc),
	)
}

// End restores original GC settings and triggers a collection.
func (gc *GCController) End() {
	if !gc.active {
		return
	}
	runtime.ReadMemStats(&gc.endStats)

	gcGlobalMu.Lock()
	if gcActiveDepth > 0 {
		gcActiveDepth--
	}
	last := gcActiveDepth == 0
	if last {
		// Last active controller: no sibling is still computing, so it is safe
		// to restore the original GOGC and soft memory limit (restoring, not
		// zeroing, so a pre-existing GOMEMLIMIT survives).
		debug.SetGCPercent(gcSavedPercent)
		debug.SetMemoryLimit(gcSavedMemLimit)
	}
	gcGlobalMu.Unlock()

	if last {
		runtime.GC()
	}

	gc.logger.LogAttrs(context.Background(), slog.LevelDebug, "gc re-enabled",
		slog.String("mode", string(gc.mode)),
		slog.Uint64("heap_alloc_bytes", gc.endStats.HeapAlloc),
		slog.Uint64("total_alloc_bytes", gc.endStats.TotalAlloc-gc.startStats.TotalAlloc),
		slog.Uint64("gc_cycles", uint64(gc.endStats.NumGC-gc.startStats.NumGC)),
	)
}
