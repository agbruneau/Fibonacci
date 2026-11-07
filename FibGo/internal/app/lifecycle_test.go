package app

import (
	"bytes"
	"context"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/config"
)

// Lifecycle regression tests for audit CON-01.
//
// Before this fix, signal.NotifyContext was installed in runCalculate,
// runLastDigits and runTUI but NOT in the two calibration paths, and only
// runCalculate / runLastDigits applied --timeout. So `--calibrate` and the
// `--auto-calibrate` phase ran on main's raw context: Ctrl-C reached the
// runtime's default handler and killed the process, and --timeout was ignored.
// The "Calibration interrupted" branch in internal/calibration existed but was
// unreachable from the binary.
//
// These tests go through Run (not runCalibration directly) precisely because
// the contract under test is where the lifecycle is installed, not what the
// calibration code does once it has a cancellable context.
//
// Not parallel: Run mutates process-global state (the zerolog level and the ui
// theme), so these must not overlap with tests that read either.

// isolatedProfilePath keeps calibration off the user's home directory. With an
// empty CalibrationProfile the code falls back to GetDefaultProfilePath(), i.e.
// ~/.fibcalc_calibration.json — a real file in a real home directory, written
// by a test run. The book is explicit about this (ch. 7, p. 226: never write to
// fixed locations, use t.TempDir()), and a leaked profile also makes the next
// test non-deterministic, because AutoCalibrate short-circuits on a cached
// profile before it ever looks at the context.
func isolatedProfilePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "calibration.json")
}

func newCalibrationApp(t *testing.T, timeout time.Duration) (*Application, *bytes.Buffer) {
	t.Helper()
	return &Application{
		Config: config.AppConfig{
			Calibrate:          true,
			Timeout:            timeout,
			Quiet:              true,
			CalibrationProfile: isolatedProfilePath(t),
		},
		Factory:   createMockFactory(big.NewInt(55), nil),
		ErrWriter: &bytes.Buffer{},
	}, &bytes.Buffer{}
}

// A context already canceled when Run is entered must reach the calibration
// sweep and produce the interrupted message plus exit 130, not run to
// completion and not die on a raw signal.
func TestRun_CalibrateHonorsCancellation(t *testing.T) {
	app, out := newCalibrationApp(t, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code := app.Run(ctx, out)

	if code != apperrors.ExitErrorCanceled {
		t.Errorf("exit code = %d, want %d (canceled)", code, apperrors.ExitErrorCanceled)
	}
	if got := out.String(); !strings.Contains(got, "Calibration interrupted") {
		t.Errorf("output does not report the interruption; got:\n%s", got)
	}
}

// --timeout must bound the calibration sweep, and an expired budget must map to
// the timeout exit code (2), not to the cancellation one (130).
//
// The budget is negative on purpose. context.WithTimeout with a non-positive
// duration is defined to return an already-expired context, which makes the
// assertion deterministic; a tiny positive value like 1ns schedules a timer and
// races the sweep, which with mock calculators takes microseconds — that race
// is exactly how the first draft of this test passed the wrong way. config
// .Validate rejects a non-positive --timeout for real users, so only a test
// that builds AppConfig directly can reach this.
func TestRun_CalibrateHonorsTimeout(t *testing.T) {
	app, out := newCalibrationApp(t, -1*time.Second)

	code := app.Run(context.Background(), out)

	if code != apperrors.ExitErrorTimeout {
		t.Errorf("exit code = %d, want %d (timeout)", code, apperrors.ExitErrorTimeout)
	}
	if got := out.String(); !strings.Contains(got, "Calibration interrupted") {
		t.Errorf("output does not report the interruption; got:\n%s", got)
	}
}

// The auto-calibration phase runs BEFORE runCalculate installs its deadline, so
// it now carries its own budget rather than eating into the calculation's. A
// canceled context must degrade to the caller's config and let the run proceed,
// never abort it.
func TestRun_AutoCalibrateCancellationDegradesToConfig(t *testing.T) {
	app := &Application{
		Config: config.AppConfig{
			AutoCalibrate:      true,
			N:                  100,
			Algo:               "fast",
			Timeout:            time.Minute,
			Quiet:              true,
			CalibrationProfile: isolatedProfilePath(t),
		},
		Factory:   createMockFactory(big.NewInt(55), nil),
		ErrWriter: &bytes.Buffer{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	before := app.Config
	got := app.runAutoCalibrationIfEnabled(ctx, &bytes.Buffer{})

	if got != before {
		t.Errorf("config changed under a canceled context:\n got  %+v\n want %+v", got, before)
	}
}

// Completion generation is deliberately above the signal root in Run: it writes
// a script and returns. A canceled context must not turn it into a failure.
func TestRun_CompletionIgnoresCancellation(t *testing.T) {
	app := &Application{
		Config:    config.AppConfig{Completion: "bash"},
		Factory:   createMockFactory(big.NewInt(55), nil),
		ErrWriter: &bytes.Buffer{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out bytes.Buffer
	if code := app.Run(ctx, &out); code != apperrors.ExitSuccess {
		t.Errorf("exit code = %d, want %d", code, apperrors.ExitSuccess)
	}
	if out.Len() == 0 {
		t.Error("no completion script written")
	}
}

// Guard for the invariant this fix establishes: exactly one signal root in the
// package, in Run. A second NotifyContext would mean a mode is either wrapping
// twice or, worse, that the duplication has crept back and a new mode was
// added without one.
func TestSignalRootIsUniqueInPackage(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	found := make([]string, 0, 1)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, line := range strings.Split(string(src), "\n") {
			// Skip prose: the surrounding comments explain the invariant and
			// naturally mention the function by name.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if strings.Contains(line, "signal.NotifyContext") {
				found = append(found, name+": "+strings.TrimSpace(line))
			}
		}
	}

	if len(found) != 1 {
		t.Errorf("want exactly 1 signal.NotifyContext call in package app, found %d:\n%s",
			len(found), strings.Join(found, "\n"))
	}
}
