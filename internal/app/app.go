package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"sync"
	"syscall"

	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/cli/completion"
	"github.com/agbruneau/FibGo/internal/config"
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/fibonacci/threshold"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/tui"
	"github.com/agbruneau/FibGo/internal/ui"
	"github.com/rs/zerolog"
)

// Application represents the fibcalc application instance.
type Application struct {
	Config    config.AppConfig
	Factory   fibonacci.CalculatorFactory
	ErrWriter io.Writer
}

// AppOption configures an Application during construction.
type AppOption func(*Application)

// WithFactory sets a custom CalculatorFactory for the application.
func WithFactory(f fibonacci.CalculatorFactory) AppOption {
	return func(a *Application) { a.Factory = f }
}

// thresholdTuningOnce guards the process-wide installation of the
// config-layer tuning profile into the threshold package.
var thresholdTuningOnce sync.Once

// wireThresholdTuning realizes the A2-04 wiring contract documented in
// threshold/manager.go and config/doc.go: translate
// config.DefaultThresholdTuning into a threshold.Tuning and install it
// exactly once, at startup, before any DynamicThresholdManager is
// constructed (single-writer-before-use). Before this call existed the
// contract was documented but never executed, so changes to
// config.DefaultThresholdTuning silently had no effect on the dynamic
// threshold manager. sync.Once keeps concurrent New calls (parallel
// tests) race-free on the unsynchronized package-level knobs.
func wireThresholdTuning() {
	thresholdTuningOnce.Do(func() {
		p := config.DefaultThresholdTuning
		threshold.SetTuning(threshold.Tuning{
			FFTSpeedupThreshold:      p.FFTSpeedupThreshold,
			ParallelSpeedupThreshold: p.ParallelSpeedupThreshold,
			HysteresisMargin:         p.HysteresisMargin,
			MinFFTThreshold:          p.MinFFTThreshold,
			MinParallelThreshold:     p.MinParallelThreshold,
		})
	})
}

// New creates a new Application instance by parsing command-line arguments.
func New(args []string, errWriter io.Writer, opts ...AppOption) (*Application, error) {
	wireThresholdTuning()

	app := &Application{ErrWriter: errWriter}
	for _, opt := range opts {
		opt(app)
	}
	if app.Factory == nil {
		app.Factory = fibonacci.NewDefaultFactory()
	}

	factory := app.Factory
	availableAlgos := factory.List()

	programName := "fibcalc"
	var cmdArgs []string
	if len(args) > 0 {
		programName = args[0]
		cmdArgs = args[1:]
	}

	cfg, err := config.ParseConfig(programName, cmdArgs, errWriter, availableAlgos)
	if err != nil {
		return nil, err
	}

	if cfgWithProfile, loaded := calibration.LoadCachedCalibration(cfg, cfg.CalibrationProfile); loaded && cfgWithProfile.Validate(availableAlgos) == nil {
		cfg = cfgWithProfile
	} else {
		cfg = config.ApplyAdaptiveThresholds(cfg)
	}

	app.Config = cfg
	return app, nil
}

// Run executes the application based on the configured mode and returns the
// POSIX exit code (internal/errors.Exit*), which main hands to os.Exit.
func (a *Application) Run(ctx context.Context, out io.Writer) int {
	if a.Config.Completion != "" {
		return a.runCompletion(out)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ui.InitTheme(a.Config.Quiet || a.Config.MachineOutput)

	if a.Config.Calibrate {
		return a.runCalibration(ctx, out)
	}

	a.Config = a.runAutoCalibrationIfEnabled(ctx, out)

	if a.Config.TUI {
		return a.runTUI(ctx, out)
	}

	return a.runCalculate(ctx, out)
}

// runCompletion generates shell completion scripts.
func (a *Application) runCompletion(out io.Writer) int {
	if err := completion.Generate(out, a.Config.Completion, a.Factory.List()); err != nil {
		fmt.Fprintf(a.ErrWriter, "Error generating completion: %v\n", err)
		return apperrors.ExitErrorConfig
	}
	return apperrors.ExitSuccess
}

// runCalibration runs the full calibration mode.
func (a *Application) runCalibration(ctx context.Context, out io.Writer) int {
	return calibration.RunCalibration(ctx, out, a.Factory.GetAll(), a.Config.CalibrationProfile, cli.DisplayProgress, cli.CLIColorProvider{})
}

// runAutoCalibrationIfEnabled runs auto-calibration if enabled.
func (a *Application) runAutoCalibrationIfEnabled(ctx context.Context, out io.Writer) config.AppConfig {
	if a.Config.AutoCalibrate {
		if updated, ok := calibration.AutoCalibrate(ctx, a.Config, out, a.Factory.GetAll()); ok {
			return updated
		}
	}
	return a.Config
}

// runTUI launches the interactive TUI dashboard.
//
// --memory-limit is validated here (audit.md APP-07): Validate() already
// rejects --tui combined with --last-digits or --output, but the TUI always
// computes the full N, so its memory budget must still be checked before
// launch — runCalculate's validateMemoryBudget call is on a path this mode
// never reaches.
//
// The timeout budget is applied per generation inside the TUI itself (in
// NewModel/handleReset, APP-05) rather than once here: a restart must get a
// fresh full budget instead of inheriting a single absolute deadline set at
// session start. signal.NotifyContext still wraps the parent context so
// SIGINT/SIGTERM cancel every generation uniformly.
func (a *Application) runTUI(ctx context.Context, out io.Writer) int {
	if code := a.validateMemoryBudget(out); code != apperrors.ExitSuccess {
		return code
	}

	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	calculatorsToRun := orchestration.GetCalculatorsToRun(a.Config.Algo, a.Factory)
	return tui.Run(ctx, calculatorsToRun, a.Config, Version, a.ErrWriter)
}

// IsHelpError checks if the error is a help flag error (--help was used).
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
