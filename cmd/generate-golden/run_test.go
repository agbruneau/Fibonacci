package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenTargets mirrors the targets slice inside run(). The duplication is a
// deliberate change detector: the committed golden corpus
// (internal/fibonacci/testdata/fibonacci_golden.json) is immutable without an
// ADR (CLAUDE.md, ADR-0004 B5), so any change to the generator's target set
// must fail loudly here instead of silently altering the next regeneration.
var goldenTargets = []uint64{
	0, 1, 2, 3, 4, 5, 10, 20, 50, 92, 93, 94, 100,
	128, 256, 512, 1000, 1024,
	2000, 2048, 5000, 8192, 10000,
	50_000, 100_000, 200_000,
}

// runWithArgs invokes run() with a controlled command line.
//
// run() reads the process-global flag.CommandLine and os.Args, and prints
// progress to os.Stdout. Tests using this helper must therefore NOT call
// t.Parallel(): all three globals are swapped for the duration of the call
// and restored via t.Cleanup. A fresh FlagSet is installed on every call so
// the "out" flag can be re-registered across successive run() invocations
// without a "flag redefined" panic.
func runWithArgs(t *testing.T, args ...string) error {
	t.Helper()

	prevArgs := os.Args
	prevCommandLine := flag.CommandLine
	prevStdout := os.Stdout

	// Silence the generator's progress output to keep test logs readable.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("opening %s: %v", os.DevNull, err)
	}

	// ContinueOnError (not ExitOnError) so a parse problem can never
	// os.Exit the test binary.
	flag.CommandLine = flag.NewFlagSet("generate-golden", flag.ContinueOnError)
	os.Args = append([]string{"generate-golden"}, args...)
	os.Stdout = devNull

	t.Cleanup(func() {
		os.Args = prevArgs
		flag.CommandLine = prevCommandLine
		os.Stdout = prevStdout
		_ = devNull.Close()
	})

	return run()
}

// TestRun_WritesGoldenFile exercises the full success path of run() against a
// temporary directory and validates the serialized output: target set and
// ordering, oracle values, JSON wire format, and O_TRUNC semantics.
func TestRun_WritesGoldenFile(t *testing.T) {
	// Serial on purpose: runWithArgs swaps process-global state.
	outDir := t.TempDir()
	filename := filepath.Join(outDir, "fibonacci_golden.json")

	// Pre-seed a stale file larger than the expected output (~80 KiB): if
	// run() ever stops truncating on open, trailing garbage survives and
	// the strict Unmarshal below fails.
	if err := os.WriteFile(filename, []byte(strings.Repeat("x", 1<<20)), 0o600); err != nil {
		t.Fatalf("seeding stale golden file: %v", err)
	}

	if err := runWithArgs(t, "-out="+outDir); err != nil {
		t.Fatalf("run() = %v, want nil", err)
	}

	raw, err := os.ReadFile(filename) // #nosec G304 -- path rooted in t.TempDir
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}

	// Pin the wire format consumed by internal/fibonacci's golden test:
	// two-space indentation, lowercase "n"/"result" JSON tags, field order.
	// A tag rename would stay self-consistent through GoldenData round-trips
	// (and so escape the Unmarshal check below), but not this prefix check.
	wantPrefix := "[\n  {\n    \"n\": 0,\n    \"result\": \"0\"\n  },\n"
	if !strings.HasPrefix(string(raw), wantPrefix) {
		t.Errorf("output prefix = %q, want %q", truncateForLog(string(raw), len(wantPrefix)), wantPrefix)
	}

	// Unmarshal (not a streaming decode) so any non-whitespace bytes after
	// the JSON document fail the test.
	var got []GoldenData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("generated file is not a single valid JSON document: %v", err)
	}

	if len(got) != len(goldenTargets) {
		t.Fatalf("got %d entries, want %d", len(got), len(goldenTargets))
	}
	for i, entry := range got {
		if entry.N != goldenTargets[i] {
			t.Errorf("entry %d: N = %d, want %d", i, entry.N, goldenTargets[i])
			continue
		}
		// Serialization fidelity: every persisted value must match the
		// in-process oracle (fibBig itself is pinned by TestFibBig).
		if want := fibBig(entry.N).String(); entry.Result != want {
			t.Errorf("entry %d: Result for F(%d) does not match the fibBig oracle", i, entry.N)
		}
		// Defense in depth for low indices: hand-checked constants.
		if want, ok := knownFibValues[entry.N]; ok && entry.Result != want {
			t.Errorf("entry %d: F(%d) = %s, want %s", i, entry.N, entry.Result, want)
		}
	}
}

// TestRun_ErrorWhenOutDirIsFile covers the MkdirAll failure branch: the -out
// path exists but is a regular file, so the directory cannot be created.
func TestRun_ErrorWhenOutDirIsFile(t *testing.T) {
	// Serial on purpose: runWithArgs swaps process-global state.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("creating blocker file: %v", err)
	}

	err := runWithArgs(t, "-out="+blocker)
	if err == nil {
		t.Fatal("run() = nil, want directory creation error")
	}
	if !strings.Contains(err.Error(), "creating output directory") {
		t.Errorf("error %q does not wrap the mkdir failure context", err)
	}
}

// TestRun_ErrorWhenGoldenPathIsDirectory covers the OpenFile failure branch:
// the output filename is occupied by a directory, so the file cannot be
// created or truncated.
func TestRun_ErrorWhenGoldenPathIsDirectory(t *testing.T) {
	// Serial on purpose: runWithArgs swaps process-global state.
	outDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(outDir, "fibonacci_golden.json"), 0o700); err != nil {
		t.Fatalf("creating blocking directory: %v", err)
	}

	err := runWithArgs(t, "-out="+outDir)
	if err == nil {
		t.Fatal("run() = nil, want file creation error")
	}
	if !strings.Contains(err.Error(), "creating output file") {
		t.Errorf("error %q does not wrap the open failure context", err)
	}
}

// truncateForLog bounds failure output so an 80 KiB document does not flood
// the test log.
func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
