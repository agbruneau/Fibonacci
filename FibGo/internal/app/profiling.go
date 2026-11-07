package app

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
)

// startProfiling begins CPU profiling when --cpuprofile is set and returns a
// stop function that ends it and, if --memprofile is set, writes a heap profile.
//
// It closes audit OBS-02. pprof was reachable only through `go test -bench`, so
// a user who saw an unexpected slowdown at their own n had no way to profile the
// binary that produced it — the one place the book puts profiling
// (ch. 8, p. 256-257).
//
// The returned function is always safe to call, even when neither flag is set
// and even after a failure, so the caller needs no bookkeeping of its own.
// Failures are reported on errWriter and do not abort the calculation: a profile
// that cannot be written is a lost diagnostic, not a wrong answer.
func (a *Application) startProfiling(errWriter io.Writer) (stop func()) {
	var cpuFile *os.File

	if path := a.Config.CPUProfile; path != "" {
		// The path is the user's own --cpuprofile argument; gosec's G304 is
		// excluded repository-wide in .golangci.yml, so no annotation is needed.
		f, err := os.Create(path)
		switch {
		case err != nil:
			fmt.Fprintf(errWriter, "cannot create CPU profile %s: %v\n", path, err)
		default:
			if err := pprof.StartCPUProfile(f); err != nil {
				fmt.Fprintf(errWriter, "cannot start CPU profile: %v\n", err)
				closeProfileFile(errWriter, f, path)
			} else {
				cpuFile = f
			}
		}
	}

	return func() {
		if cpuFile != nil {
			pprof.StopCPUProfile()
			closeProfileFile(errWriter, cpuFile, a.Config.CPUProfile)
		}
		a.writeMemProfile(errWriter)
	}
}

// writeMemProfile writes a heap profile to --memprofile, if set.
//
// runtime.GC() first, as the pprof documentation requires: without it the
// profile reports whatever the allocator happened to be holding rather than
// what is genuinely live.
func (a *Application) writeMemProfile(errWriter io.Writer) {
	path := a.Config.MemProfile
	if path == "" {
		return
	}

	// The path is the user's own --memprofile argument; see startProfiling.
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(errWriter, "cannot create memory profile %s: %v\n", path, err)
		return
	}
	defer closeProfileFile(errWriter, f, path)

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintf(errWriter, "cannot write memory profile: %v\n", err)
	}
}

// closeProfileFile closes f and reports a close failure, which for a profile
// means a truncated file — worth saying, since the user will otherwise open it
// in pprof and see a parse error with no explanation.
func closeProfileFile(errWriter io.Writer, f *os.File, path string) {
	if err := f.Close(); err != nil {
		fmt.Fprintf(errWriter, "cannot close profile %s: %v\n", path, err)
	}
}
