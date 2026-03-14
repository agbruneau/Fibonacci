package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
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
	if c.Threshold < 0 {
		return apperrors.NewConfigError("parallelism threshold cannot be negative: %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return apperrors.NewConfigError("FFT threshold cannot be negative: %d", c.FFTThreshold)
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
	algoHelp := fmt.Sprintf("Algorithm to use: 'all' (default) or one of [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", DefaultN, "Index n of the Fibonacci number to calculate.")
	fs.BoolVar(&config.Verbose, "v", false, "Display the full value of the result (can be very long).")
	fs.BoolVar(&config.Verbose, "verbose", false, "Alias for -v.")
	fs.BoolVar(&config.Details, "d", false, "Display performance details and result metadata.")
	fs.BoolVar(&config.Details, "details", false, "Alias for -d.")
	fs.DurationVar(&config.Timeout, "timeout", DefaultTimeout, "Maximum execution time for the calculation.")
	fs.StringVar(&config.Algo, "algo", DefaultAlgo, algoHelp)
	fs.IntVar(&config.Threshold, "threshold", 0, "Threshold (in bits) for activating parallelism in multiplications (0 for auto).")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 0, "Threshold (in bits) to enable FFT multiplication (0 for auto).")
	fs.IntVar(&config.StrassenThreshold, "strassen-threshold", 0, "Threshold (in bits) to switch to Strassen's algorithm in matrix multiplication (0 for auto).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Runs calibration mode to determine the optimal parallelism threshold.")
	fs.BoolVar(&config.AutoCalibrate, "auto-calibrate", false, "Enables quick automatic calibration at startup (may increase loading time).")
	fs.StringVar(&config.CalibrationProfile, "calibration-profile", "", "Path to calibration profile file (default: ~/.fibcalc_calibration.json).")
	// New CLI enhancement flags
	fs.StringVar(&config.OutputFile, "output", "", "Output file path for the result.")
	fs.StringVar(&config.OutputFile, "o", "", "Output file path (shorthand).")
	fs.BoolVar(&config.Quiet, "quiet", false, "Quiet mode - minimal output for scripts.")
	fs.BoolVar(&config.Quiet, "q", false, "Quiet mode (shorthand).")
	fs.StringVar(&config.Completion, "completion", "", "Generate shell completion script (bash, zsh, fish, powershell).")
	fs.BoolVar(&config.ShowValue, "calculate", false, "Display the calculated value (disabled by default).")
	fs.BoolVar(&config.ShowValue, "c", false, "Display the calculated value (shorthand).")
	fs.BoolVar(&config.TUI, "tui", false, "Launch interactive TUI dashboard.")
	fs.IntVar(&config.LastDigits, "last-digits", 0, "Compute only the last K decimal digits (uses O(K) memory).")
	fs.StringVar(&config.MemoryLimit, "memory-limit", "", "Maximum memory budget (e.g., 8G, 512M). Warns if estimate exceeds limit.")
	fs.StringVar(&config.GCControl, "gc-control", "auto", "GC control during calculation (auto, aggressive, disabled).")
	setCustomUsage(fs)

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}

	// Apply environment variable overrides for flags not explicitly set
	applyEnvOverrides(&config, fs)

	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Configuration error:", err)
		fs.Usage()
		return AppConfig{}, errors.New("invalid configuration")
	}
	return config, nil
}
