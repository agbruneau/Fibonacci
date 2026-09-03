package config

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
)

const (
	// EnvPrefix is the prefix for all environment variables used by fibcalc.
	// Environment variables provide an alternative to CLI flags for configuration,
	// following the 12-Factor App methodology.
	EnvPrefix = "FIBCALC_"
)

// Default configuration values.
// These can be overridden via command-line flags or environment variables.
const (
	// DefaultN is the default Fibonacci index to calculate.
	DefaultN uint64 = 100_000_000
	// DefaultTimeout is the default calculation timeout.
	DefaultTimeout = 5 * time.Minute
	// DefaultAlgo is the default algorithm selection.
	DefaultAlgo = "all"
)

// ThresholdDisabled is the value that turns a threshold OFF, as opposed to 0
// which means "auto" (fill from the hardware heuristic, then from the static
// default in fibonacci/constants.go).
//
// It is the single source of truth for a contract that was previously an
// unnamed internal sentinel and a validation bug (audit H-02). The calibration
// candidate lists have used -1 as their genuine no-parallelism / no-FFT
// baseline since FIB-02, because normalizeOptions only substitutes a default
// for ==0 and a 0 candidate therefore silently re-measured the default. When
// that baseline won, calibration persisted -1 into the profile — a value
// Validate then rejected, so app.New discarded the whole profile without a
// word on every subsequent start. Accepting -1 is what makes the calibration
// result usable on hosts where sequential (or non-FFT) really is fastest.
//
// It applies to Threshold and FFTThreshold ONLY. Every consumer of those two
// gates on `> 0` — fastdoubling.go and matrix_framework.go for parallelism,
// fft.go and strategy.go for FFT — so a negative value reads as "never".
// StrassenThreshold has no such gate: multiplyMatrices compares
// `maxBitLen <= strassenThreshold`, so a negative value would force Strassen
// always, the opposite of disabling it. It keeps rejecting anything below 0,
// and no candidate generator produces -1 for it.
const ThresholdDisabled = -1

// MaxLastDigits bounds the K accepted by --last-digits. The path is O(K)
// memory (10^K as a big.Int, ~K*3.32 bits) and, unlike the full-N path, is
// never checked against --memory-limit, so K itself must stay bounded whether
// or not a limit was set (audit SEC-03).
//
// It lives here rather than in internal/app (audit M-02) so the bound applies
// at parse time, on every mode, instead of only on the one path that reaches
// runLastDigits. The app-level guard is kept as defense in depth for callers
// that build an Application programmatically without going through Validate.
const MaxLastDigits = 10_000_000

// AppConfig aggregates the application's configuration parameters, parsed from
// command-line flags. It encapsulates all settings that control the execution,
// from the Fibonacci index to calculate, to performance-tuning parameters.
type AppConfig struct {
	// N is the index of the Fibonacci number to be calculated.
	N uint64
	// Verbose, if true, instructs the application to display the full calculated number.
	Verbose bool
	// Details, if true, provides a detailed report including performance metrics.
	Details bool
	// Timeout sets the maximum duration for the calculation.
	Timeout time.Duration
	// Algo specifies the algorithm to use ("all", "fast", "matrix", etc.).
	Algo string
	// Threshold determines the bit size at which multiplications are parallelized.
	Threshold int
	// FFTThreshold is the bit size threshold for using FFT-based multiplication.
	FFTThreshold int
	// StrassenThreshold controls when matrix multiplication switches to Strassen.
	StrassenThreshold int
	// Calibrate, if true, runs the application in calibration mode to find the
	// optimal parallelism threshold.
	Calibrate bool
	// AutoCalibrate, if true, runs a short automatic calibration at startup to
	// refine Threshold and FFTThreshold for the current machine.
	AutoCalibrate bool
	// CalibrationProfile is the path to a calibration profile file.
	// If set, the application will load/save calibration results from/to this file.
	// If empty, uses the default path (~/.fibcalc_calibration.json).
	CalibrationProfile string
	// OutputFile, if specified, saves the result to this file path.
	OutputFile string
	// Quiet mode - minimal output for scripting purposes.
	// Suppresses progress bars, banners, and informational messages.
	Quiet bool
	// MachineOutput requests predictable, non-ANSI CLI output for pipelines
	// (stderr and themed text). Combine with -quiet for minimal stdout.
	MachineOutput bool
	// Completion, if set, generates shell completion script for the specified shell.
	// Valid values are: "bash", "zsh", "fish", "powershell".
	Completion string
	// ShowValue, if true, displays the calculated Fibonacci value. Set with -c/--calculate.
	ShowValue bool
	// TUI, if true, launches the interactive TUI dashboard instead of CLI mode.
	TUI bool
	// LastDigits, if > 0, computes only the last K decimal digits of F(N).
	// Uses O(K) memory via modular arithmetic.
	LastDigits int
	// MemoryLimit, if set, specifies the maximum memory budget for calculation.
	// Accepts human-readable formats like "8G", "512M", "1024K".
	// The application warns and exits if the estimated memory exceeds this limit.
	MemoryLimit string
	// GCControl sets the GC control mode ("auto", "aggressive", "disabled").
	GCControl string
}

// Validate checks the semantic consistency of the configuration parameters.
// It ensures that numerical values are within valid ranges and that the chosen
// algorithm is supported.
//
// Parameters:
//   - availableAlgos: A slice of strings listing the valid algorithm names
//     (e.g., ["fast", "matrix"]).
//
// Returns:
//   - error: An error of type ConfigError if the configuration is invalid,
//     nil otherwise.
func (c AppConfig) Validate(availableAlgos []string) error {
	if c.Timeout <= 0 {
		return apperrors.NewConfigError("timeout value must be strictly positive")
	}
	// ThresholdDisabled (-1) is accepted for the two thresholds whose consumers
	// gate on `> 0`; see its doc comment for why Strassen is excluded.
	if c.Threshold < ThresholdDisabled {
		return apperrors.NewConfigError("parallelism threshold must be >= %d (%d disables parallelism, 0 is auto): %d", ThresholdDisabled, ThresholdDisabled, c.Threshold)
	}
	if c.FFTThreshold < ThresholdDisabled {
		return apperrors.NewConfigError("FFT threshold must be >= %d (%d disables FFT, 0 is auto): %d", ThresholdDisabled, ThresholdDisabled, c.FFTThreshold)
	}
	if c.StrassenThreshold < 0 {
		return apperrors.NewConfigError("Strassen threshold cannot be negative: %d", c.StrassenThreshold)
	}
	if c.LastDigits < 0 {
		return apperrors.NewConfigError("last-digits cannot be negative: %d (0 disables, >0 computes the last K digits)", c.LastDigits)
	}
	if c.LastDigits > MaxLastDigits {
		return apperrors.NewConfigError("last-digits %d exceeds the maximum of %d digits", c.LastDigits, MaxLastDigits)
	}
	// Parse --memory-limit here so a malformed value is a configuration error
	// on every mode. It used to be parsed only by app.validateMemoryBudget,
	// which the --last-digits and --calibrate paths never reach, so
	// `--last-digits 5 --memory-limit 4GB` ran to completion without a word
	// about the unusable limit (audit M-02). The parsed value is discarded:
	// ValidateMemoryBudget re-parses it when it actually needs the number.
	if c.MemoryLimit != "" {
		if _, err := memory.ParseMemoryLimit(c.MemoryLimit); err != nil {
			return apperrors.NewConfigError("invalid memory limit %q: %v", c.MemoryLimit, err)
		}
	}
	if c.TUI && c.LastDigits > 0 {
		return apperrors.NewConfigError("--tui is incompatible with --last-digits: the TUI dashboard always computes the full value")
	}
	if c.TUI && c.OutputFile != "" {
		return apperrors.NewConfigError("--tui is incompatible with --output: the TUI dashboard does not save results to a file")
	}
	switch c.GCControl {
	case "", string(memory.GCModeAuto), string(memory.GCModeAggressive), string(memory.GCModeDisabled):
		// valid
	default:
		return apperrors.NewConfigError("unrecognized gc-control mode: '%s'. Valid modes are: auto, aggressive, disabled", c.GCControl)
	}
	switch c.Completion {
	case "", "bash", "zsh", "fish", "powershell":
		// valid
	default:
		return apperrors.NewConfigError("unrecognized completion shell: '%s'. Valid shells are: bash, zsh, fish, powershell", c.Completion)
	}
	isAlgoAvailable := false
	for _, a := range availableAlgos {
		if a == c.Algo {
			isAlgoAvailable = true
			break
		}
	}
	if c.Algo != "all" && !isAlgoAvailable {
		return apperrors.NewConfigError("unrecognized algorithm: '%s'. Valid algorithms are: 'all' or [%s]", c.Algo, strings.Join(availableAlgos, ", "))
	}
	return nil
}

// registerFlags binds all CLI flags to the given AppConfig on the provided
// FlagSet. It is the single source of truth for flag names and defaults, and
// is shared between ParseConfig and integrity tests that verify env-override
// flag references stay in sync with the actual flag set.
func registerFlags(fs *flag.FlagSet, config *AppConfig, availableAlgos []string) {
	algoHelp := fmt.Sprintf("Algorithm to use: 'all' (default) or one of [%s].", strings.Join(availableAlgos, ", "))

	fs.Uint64Var(&config.N, "n", DefaultN, "Index n of the Fibonacci number to calculate.")
	fs.BoolVar(&config.Verbose, "v", false, "Display the full value of the result (can be very long).")
	fs.BoolVar(&config.Verbose, "verbose", false, "Alias for -v.")
	fs.BoolVar(&config.Details, "d", false, "Display performance details and result metadata.")
	fs.BoolVar(&config.Details, "details", false, "Alias for -d.")
	fs.DurationVar(&config.Timeout, "timeout", DefaultTimeout, "Maximum execution time for the calculation.")
	fs.StringVar(&config.Algo, "algo", DefaultAlgo, algoHelp)
	fs.IntVar(&config.Threshold, "threshold", 0, "Threshold (in bits) for activating parallelism in multiplications (0 for auto, -1 to disable).")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 0, "Threshold (in bits) to enable FFT multiplication (0 for auto, -1 to disable).")
	fs.IntVar(&config.StrassenThreshold, "strassen-threshold", 0, "Threshold (in bits) to switch to Strassen's algorithm in matrix multiplication (0 for auto).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Runs calibration mode to determine the optimal parallelism threshold.")
	fs.BoolVar(&config.AutoCalibrate, "auto-calibrate", false, "Enables quick automatic calibration at startup (may increase loading time).")
	fs.StringVar(&config.CalibrationProfile, "calibration-profile", "", "Path to calibration profile file (default: ~/.fibcalc_calibration.json).")
	// New CLI enhancement flags
	fs.StringVar(&config.OutputFile, "output", "", "Output file path for the result.")
	fs.StringVar(&config.OutputFile, "o", "", "Output file path (shorthand).")
	fs.BoolVar(&config.Quiet, "quiet", false, "Quiet mode - minimal output for scripts.")
	fs.BoolVar(&config.Quiet, "q", false, "Quiet mode (shorthand).")
	fs.BoolVar(&config.MachineOutput, "machine", false, "Machine-readable output: no ANSI colors (for scripts and CI).")
	fs.StringVar(&config.Completion, "completion", "", "Generate shell completion script (bash, zsh, fish, powershell).")
	fs.BoolVar(&config.ShowValue, "calculate", false, "Display the calculated value (disabled by default).")
	fs.BoolVar(&config.ShowValue, "c", false, "Display the calculated value (shorthand).")
	fs.BoolVar(&config.TUI, "tui", false, "Launch interactive TUI dashboard.")
	fs.IntVar(&config.LastDigits, "last-digits", 0, "Compute only the last K decimal digits (uses O(K) memory).")
	fs.StringVar(&config.MemoryLimit, "memory-limit", "", "Maximum memory budget (e.g., 8G, 512M). Aborts with a config error if the estimate exceeds it.")
	fs.StringVar(&config.GCControl, "gc-control", "auto", "GC control during calculation (auto, aggressive, disabled).")
}

// FlagNames returns the names of every CLI flag registered by registerFlags.
// It is the canonical flag list and exists so shell-completion generation (and
// its sync test) can verify it never drifts from the real flag set. Note: it
// does not include --version/-V or --help/-h, which are handled outside the
// FlagSet (HasVersionFlag and flag.ErrHelp respectively).
func FlagNames() []string {
	fs := flag.NewFlagSet("fibcalc", flag.ContinueOnError)
	registerFlags(fs, &AppConfig{}, nil)
	var names []string
	fs.VisitAll(func(f *flag.Flag) { names = append(names, f.Name) })
	return names
}

// ParseConfig parses the command-line arguments and populates an AppConfig
// struct. It defines all the command-line flags, sets their default values, and
// handles the parsing process. After parsing, it performs validation on the
// resulting configuration.
//
// The function is designed to be testable by allowing the input arguments and
// output writer to be specified.
//
// Parameters:
//   - programName: The name of the program, used in the usage message.
//   - args: A slice of strings representing the command-line arguments
//     (typically os.Args[1:]).
//   - errorWriter: An io.Writer where parsing errors and usage information
//     will be printed.
//   - availableAlgos: A slice of valid algorithm names for validation.
//
// Returns:
//   - AppConfig: The populated configuration struct.
//   - error: An error if flag parsing fails or validation fails.
func ParseConfig(programName string, args []string, errorWriter io.Writer, availableAlgos []string) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)

	config := AppConfig{}
	registerFlags(fs, &config, availableAlgos)
	setCustomUsage(fs)

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}

	// Apply environment variable overrides for flags not explicitly set.
	// A malformed, explicitly-set override is a hard configuration error
	// rather than a silent fallback to the default.
	if err := applyEnvOverrides(&config, fs); err != nil {
		fmt.Fprintln(errorWriter, "Configuration error:", err)
		fs.Usage()
		return AppConfig{}, err
	}

	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Configuration error:", err)
		fs.Usage()
		return AppConfig{}, err
	}
	return config, nil
}
