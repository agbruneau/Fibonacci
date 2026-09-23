package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/agbruneau/FibGo/internal/apperrors"
	"github.com/agbruneau/FibGo/internal/calibration"
	"github.com/agbruneau/FibGo/internal/cli"
	"github.com/agbruneau/FibGo/internal/cli/completion"
	"github.com/agbruneau/FibGo/internal/config"
	"github.com/agbruneau/FibGo/internal/fibonacci"
	"github.com/agbruneau/FibGo/internal/orchestration"
	"github.com/agbruneau/FibGo/internal/tui"
	"github.com/agbruneau/FibGo/internal/ui"
)

// CalculatorRegistry is what the application layer needs from a calculator
// registry: the two lookups orchestration resolves --algo with, plus the full
// map that calibration sweeps.
//
// Consumer-defined, like orchestration.CalculatorSource it embeds (audit
// API-01). fibonacci.CalculatorFactory, which this replaces, also declared
// Create and Register — neither of which any caller outside the fibonacci
// package's own tests ever used.
type CalculatorRegistry interface {
	orchestration.CalculatorSource

	// GetAll returns every registered calculator, keyed by name. Calibration
	// needs the whole set, not one lookup at a time.
	GetAll() map[string]fibonacci.Calculator
}

// Application represents the fibcalc application instance.
type Application struct {
	Config    config.AppConfig
	Factory   CalculatorRegistry
	ErrWriter io.Writer

	// logger is the diagnostic logger for this run, nil when --log-level is
	// off. Run installs it; the calculation paths pass it down through
	// fibonacci.Options.
	logger *slog.Logger
}

// AppOption configures an Application during construction.
type AppOption func(*Application)

// WithFactory sets a custom calculator registry for the application.
func WithFactory(f CalculatorRegistry) AppOption {
	return func(a *Application) { a.Factory = f }
}

// New creates a new Application instance by parsing command-line arguments.
func New(args []string, errWriter io.Writer, opts ...AppOption) (*Application, error) {
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
// POSIX exit code (internal/apperrors.Exit*), which main hands to os.Exit.
//
// Signal handling is installed HERE, once, for every mode that computes (audit
// CON-01). It used to be duplicated in runCalculate, runLastDigits and runTUI,
// which left two modes uncovered: `--calibrate` and the `--auto-calibrate`
// phase ran on main's raw context, so Ctrl-C hit the runtime's default handler
// and killed the process outright. The "Calibration interrupted" branch in
// internal/calibration was unreachable from the binary, and so was the timeout
// path — those modes ignored --timeout entirely.
//
// Completion generation is deliberately above the signal root: it writes a
// script and returns, with nothing to interrupt.
func (a *Application) Run(ctx context.Context, out io.Writer) int {
	if a.Config.Completion != "" {
		return a.runCompletion(out)
	}

	// The diagnostic logger is built once, here, and handed to the domain
	// through fibonacci.Options (audit OBS-01). What stood here before was
	// zerolog.SetGlobalLevel(zerolog.InfoLevel) — a process-wide mutation whose
	// only effect was to silence the one emitter that was not already wired to
	// a no-op.
	a.logger = a.newDiagnosticLogger()
	ui.InitTheme(a.Config.Quiet || a.Config.MachineOutput)

	// --cpuprofile / --memprofile cover every computing mode, and the deferred
	// stop runs on the error paths too (audit OBS-02).
	defer a.startProfiling(a.ErrWriter)()

	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

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
//
// --timeout bounds the whole sweep (audit CON-01). It used to bound nothing
// here: the mode received main's raw context, so a sweep that stalled ran until
// the user killed it. Each pass computes F(CalibrationN) at one threshold, so
// the sweep is a sequence of calculations and the documented "maximum execution
// time for the calculation" is the honest budget for it.
func (a *Application) runCalibration(ctx context.Context, out io.Writer) int {
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()

	return calibration.RunCalibration(ctx, out, cli.NewCalibrationReporter(out), a.Factory.GetAll(), a.Config.CalibrationProfile, cli.DisplayProgress)
}

// runAutoCalibrationIfEnabled runs auto-calibration if enabled.
//
// The micro-benchmark phase gets its own timeout budget rather than eating into
// the calculation's: it runs BEFORE runCalculate installs its deadline, and a
// slow probe must not silently consume the time the user asked for the actual
// computation. AutoCalibrate degrades to the unchanged config on failure, so an
// expired budget here costs defaults, not an aborted run.
func (a *Application) runAutoCalibrationIfEnabled(ctx context.Context, out io.Writer) config.AppConfig {
	if !a.Config.AutoCalibrate {
		return a.Config
	}

	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()

	if updated, ok := calibration.AutoCalibrate(ctx, a.Config, cli.NewCalibrationReporter(out), a.Factory.GetAll()); ok {
		return updated
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
// session start. The signal root installed in Run wraps the context passed in,
// so SIGINT/SIGTERM cancel every generation uniformly.
func (a *Application) runTUI(ctx context.Context, out io.Writer) int {
	if code := a.validateMemoryBudget(out); code != apperrors.ExitSuccess {
		return code
	}

	calculatorsToRun := orchestration.GetCalculatorsToRun(a.Config.Algo, a.Factory)
	return tui.Run(ctx, calculatorsToRun, a.Config, Version, a.ErrWriter, a.logger)
}

// IsHelpError checks if the error is a help flag error (--help was used).
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
