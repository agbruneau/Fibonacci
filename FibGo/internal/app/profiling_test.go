package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agbruneau/FibGo/internal/config"
)

// Regression tests for audit OBS-02: pprof was reachable only through
// `go test -bench`, so a user who hit a slowdown at their own n could not
// profile the binary that produced it.

func TestStartProfiling_WritesBothProfiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpuPath := filepath.Join(dir, "cpu.prof")
	memPath := filepath.Join(dir, "mem.prof")

	var errOut bytes.Buffer
	app := &Application{
		Config:    config.AppConfig{CPUProfile: cpuPath, MemProfile: memPath},
		ErrWriter: &errOut,
	}

	stop := app.startProfiling(&errOut)
	// Something for the CPU profiler to sample; the content does not matter,
	// only that the file is a valid, non-empty pprof stream.
	sum := 0
	for i := 0; i < 1_000_000; i++ {
		sum += i
	}
	_ = sum
	stop()

	if errOut.Len() != 0 {
		t.Errorf("profiling reported a problem: %s", errOut.String())
	}
	for name, path := range map[string]string{"CPU": cpuPath, "memory": memPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("%s profile not written: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s profile is empty", name)
		}
	}
}

// Neither flag set: nothing is written, and the returned stop is still safe to
// call. This is the default path for every run.
func TestStartProfiling_DisabledByDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var errOut bytes.Buffer
	app := &Application{Config: config.AppConfig{}, ErrWriter: &errOut}

	app.startProfiling(&errOut)()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("profiling wrote %d file(s) with both flags unset", len(entries))
	}
	if errOut.Len() != 0 {
		t.Errorf("profiling wrote to stderr with both flags unset: %s", errOut.String())
	}
}

// An unwritable path must be reported and must not abort the run: a profile
// that cannot be written is a lost diagnostic, not a wrong answer.
func TestStartProfiling_UnwritablePathIsReportedNotFatal(t *testing.T) {
	t.Parallel()

	// A path whose parent does not exist fails os.Create on every platform.
	missing := filepath.Join(t.TempDir(), "no-such-dir", "cpu.prof")

	var errOut bytes.Buffer
	app := &Application{
		Config:    config.AppConfig{CPUProfile: missing, MemProfile: missing},
		ErrWriter: &errOut,
	}

	app.startProfiling(&errOut)()

	got := errOut.String()
	if !strings.Contains(got, "cannot create CPU profile") {
		t.Errorf("the CPU profile failure was not reported: %q", got)
	}
	if !strings.Contains(got, "cannot create memory profile") {
		t.Errorf("the memory profile failure was not reported: %q", got)
	}
}

// Only --memprofile: the heap profile is written and no CPU profile is started.
func TestStartProfiling_MemoryOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	memPath := filepath.Join(dir, "mem.prof")

	var errOut bytes.Buffer
	app := &Application{
		Config:    config.AppConfig{MemProfile: memPath},
		ErrWriter: &errOut,
	}

	app.startProfiling(&errOut)()

	if info, err := os.Stat(memPath); err != nil || info.Size() == 0 {
		t.Errorf("memory profile not written: err=%v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected only the memory profile, found %d files", len(entries))
	}
}
