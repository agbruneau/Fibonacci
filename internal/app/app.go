package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/agbru/fibcalc/internal/calibration"
	"github.com/agbru/fibcalc/internal/cli"
	"github.com/agbru/fibcalc/internal/config"
	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/orchestration"
	"github.com/agbru/fibcalc/internal/tui"
	"github.com/agbru/fibcalc/internal/ui"
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

	if cfgWithProfile, loaded := calibration.LoadCachedCalibration(cfg, cfg.CalibrationProfile); loaded {
		cfg = cfgWithProfile
	} else {
		cfg = config.ApplyAdaptiveThresholds(cfg)
	}

	app.Config = cfg
	return app, nil
}

// Run executes the application based on the configured mode.
func (a *Application) Run(ctx context.Context, out io.Writer) int {
	if a.Config.Completion != "" {
		return a.runCompletion(out)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	ui.InitTheme(false)

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
	availableAlgos := a.Factory.List()
	if err := cli.GenerateCompletion(out, a.Config.Completion, availableAlgos); err != nil {
		fmt.Fprintf(a.ErrWriter, "Error generating completion: %v\n", err)
		return apperrors.ExitErrorConfig
	}
	return apperrors.ExitSuccess
}

// runCalibration runs the full calibration mode.
func (a *Application) runCalibration(ctx context.Context, out io.Writer) int {
	return calibration.RunCalibration(ctx, out, a.Factory.GetAll(), cli.DisplayProgress, cli.CLIColorProvider{})
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
func (a *Application) runTUI(ctx context.Context, _ io.Writer) int {
	ctx, cancelTimeout := context.WithTimeout(ctx, a.Config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	calculatorsToRun := orchestration.GetCalculatorsToRun(a.Config.Algo, a.Factory)
	return tui.Run(ctx, calculatorsToRun, a.Config, Version)
}

// IsHelpError checks if the error is a help flag error (--help was used).
func IsHelpError(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
