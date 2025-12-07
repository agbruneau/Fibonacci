// Package config provides the configuration management for the fibcalc application.
// It defines the data structure for the configuration, handles the parsing of
// command-line arguments, and performs validation on the configuration values.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/agbru/fibcalc/internal/errors"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

const (
	// EnvPrefix is the prefix for all environment variables used by fibcalc.
	// Environment variables provide an alternative to CLI flags for configuration,
	// following the 12-Factor App methodology.
	EnvPrefix = "FIBCALC_"
)

// ─────────────────────────────────────────────────────────────────────────────
// Environment Variable Utilities
// ─────────────────────────────────────────────────────────────────────────────

// getEnvString returns the value of the environment variable with the given key
// (prefixed with EnvPrefix), or the default value if not set.
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvUint64 returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as uint64, or the default value if not set
// or invalid.
func getEnvUint64(key string, defaultVal uint64) uint64 {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := strconv.ParseUint(val, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// getEnvInt returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as int, or the default value if not set
// or invalid.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// getEnvBool returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as bool, or the default value if not set.
// Accepts "true", "1", "yes" as true; "false", "0", "no" as false (case-insensitive).
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return defaultVal
}

// getEnvDuration returns the value of the environment variable with the given key
// (prefixed with EnvPrefix) parsed as time.Duration, or the default value if not
// set or invalid. Accepts formats like "5m", "30s", "1h30m".
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(EnvPrefix + key); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// isFlagSet checks if a flag was explicitly set on the command line.
// This is used to determine whether to apply environment variable overrides.
func isFlagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// applyEnvOverrides applies environment variable values to the configuration
// for any flags that were not explicitly set on the command line.
// This implements the priority: CLI flags > Environment variables > Defaults.
//
// Supported environment variables:
//   - FIBCALC_N: Index of the Fibonacci number to calculate (uint64)
//   - FIBCALC_ALGO: Algorithm to use (string: fast, matrix, fft, all)
//   - FIBCALC_PORT: Port for server mode (string)
//   - FIBCALC_TIMEOUT: Calculation timeout (duration: "5m", "30s")
//   - FIBCALC_THRESHOLD: Parallelism threshold in bits (int)
//   - FIBCALC_FFT_THRESHOLD: FFT multiplication threshold in bits (int)
//   - FIBCALC_STRASSEN_THRESHOLD: Strassen algorithm threshold in bits (int)
//   - FIBCALC_SERVER: Enable server mode (bool: true/false, 1/0, yes/no)
//   - FIBCALC_JSON: Enable JSON output (bool)
//   - FIBCALC_VERBOSE: Enable verbose output (bool)
//   - FIBCALC_QUIET: Enable quiet mode (bool)
//   - FIBCALC_HEX: Enable hexadecimal output (bool)
//   - FIBCALC_INTERACTIVE: Enable interactive REPL mode (bool)
//   - FIBCALC_NO_COLOR: Disable colored output (bool)
//   - FIBCALC_OUTPUT: Output file path (string)
//   - FIBCALC_CALIBRATION_PROFILE: Path to calibration profile (string)
func applyEnvOverrides(config *AppConfig, fs *flag.FlagSet) {
	// Numeric parameters
	if !isFlagSet(fs, "n") {
		config.N = getEnvUint64("N", config.N)
	}
	if !isFlagSet(fs, "threshold") {
		config.Threshold = getEnvInt("THRESHOLD", config.Threshold)
	}
	if !isFlagSet(fs, "fft-threshold") {
		config.FFTThreshold = getEnvInt("FFT_THRESHOLD", config.FFTThreshold)
	}
	if !isFlagSet(fs, "strassen-threshold") {
		config.StrassenThreshold = getEnvInt("STRASSEN_THRESHOLD", config.StrassenThreshold)
	}

	// Duration parameters
	if !isFlagSet(fs, "timeout") {
		config.Timeout = getEnvDuration("TIMEOUT", config.Timeout)
	}

	// String parameters
	if !isFlagSet(fs, "algo") {
		config.Algo = getEnvString("ALGO", config.Algo)
	}
	if !isFlagSet(fs, "port") {
		config.Port = getEnvString("PORT", config.Port)
	}
	if !isFlagSet(fs, "output") && !isFlagSet(fs, "o") {
		config.OutputFile = getEnvString("OUTPUT", config.OutputFile)
	}
	if !isFlagSet(fs, "calibration-profile") {
		config.CalibrationProfile = getEnvString("CALIBRATION_PROFILE", config.CalibrationProfile)
	}

	// Boolean parameters
	if !isFlagSet(fs, "server") {
		config.ServerMode = getEnvBool("SERVER", config.ServerMode)
	}
	if !isFlagSet(fs, "json") {
		config.JSONOutput = getEnvBool("JSON", config.JSONOutput)
	}
	if !isFlagSet(fs, "v") {
		config.Verbose = getEnvBool("VERBOSE", config.Verbose)
	}
	if !isFlagSet(fs, "d") && !isFlagSet(fs, "details") {
		config.Details = getEnvBool("DETAILS", config.Details)
	}
	if !isFlagSet(fs, "quiet") && !isFlagSet(fs, "q") {
		config.Quiet = getEnvBool("QUIET", config.Quiet)
	}
	if !isFlagSet(fs, "hex") {
		config.HexOutput = getEnvBool("HEX", config.HexOutput)
	}
	if !isFlagSet(fs, "interactive") {
		config.Interactive = getEnvBool("INTERACTIVE", config.Interactive)
	}
	if !isFlagSet(fs, "no-color") {
		config.NoColor = getEnvBool("NO_COLOR", config.NoColor)
	}
	if !isFlagSet(fs, "calibrate") {
		config.Calibrate = getEnvBool("CALIBRATE", config.Calibrate)
	}
	if !isFlagSet(fs, "auto-calibrate") {
		config.AutoCalibrate = getEnvBool("AUTO_CALIBRATE", config.AutoCalibrate)
	}
	if !isFlagSet(fs, "calculate") && !isFlagSet(fs, "c") {
		config.Concise = getEnvBool("CALCULATE", config.Concise)
	}
}

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
	// JSONOutput, if true, outputs the result in JSON format.
	JSONOutput bool
	// ServerMode, if true, starts the application as an HTTP server.
	ServerMode bool
	// Port specifies the port to listen on in server mode.
	Port string
	// NoColor, if true, disables all color output in the CLI.
	// Also respects the NO_COLOR environment variable.
	NoColor bool

	// OutputFile, if specified, saves the result to this file path.
	OutputFile string
	// Quiet mode - minimal output for scripting purposes.
	// Suppresses progress bars, banners, and informational messages.
	Quiet bool
	// HexOutput, if true, displays the result in hexadecimal format.
	HexOutput bool
	// Interactive, if true, starts the application in REPL mode.
	Interactive bool
	// Completion, if set, generates shell completion script for the specified shell.
	// Valid values are: "bash", "zsh", "fish", "powershell".
	Completion string
	// Concise, if false (default), suppresses the display of the calculated value section.
	// Set to true with -c/--calculate to display the calculated value.
	Concise bool
}

// ToCalculationOptions converts the application configuration into
// fibonacci.Options for use by the calculators.
func (c AppConfig) ToCalculationOptions() fibonacci.Options {
	return fibonacci.Options{
		ParallelThreshold: c.Threshold,
		FFTThreshold:      c.FFTThreshold,
		StrassenThreshold: c.StrassenThreshold,
	}
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
	fs.Uint64Var(&config.N, "n", 250000000, "Index n of the Fibonacci number to calculate.")
	fs.BoolVar(&config.Verbose, "v", false, "Display the full value of the result (can be very long).")
	fs.BoolVar(&config.Details, "d", false, "Display performance details and result metadata.")
	fs.BoolVar(&config.Details, "details", false, "Alias for -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Maximum execution time for the calculation.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", 4096, "Threshold (in bits) for activating parallelism in multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 1_000_000, "Threshold (in bits) to enable FFT multiplication (0 to disable).")
	fs.IntVar(&config.StrassenThreshold, "strassen-threshold", 3072, "Threshold (in bits) to switch to Strassen's algorithm in matrix multiplication.")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Runs calibration mode to determine the optimal parallelism threshold.")
	fs.BoolVar(&config.AutoCalibrate, "auto-calibrate", false, "Enables quick automatic calibration at startup (may increase loading time).")
	fs.StringVar(&config.CalibrationProfile, "calibration-profile", "", "Path to calibration profile file (default: ~/.fibcalc_calibration.json).")
	fs.BoolVar(&config.JSONOutput, "json", false, "Output results in JSON format.")
	fs.BoolVar(&config.ServerMode, "server", false, "Start in HTTP server mode.")
	fs.StringVar(&config.Port, "port", "8080", "Port to listen on in server mode.")
	fs.BoolVar(&config.NoColor, "no-color", false, "Disable colored output (also respects NO_COLOR env var).")

	// New CLI enhancement flags
	fs.StringVar(&config.OutputFile, "output", "", "Output file path for the result.")
	fs.StringVar(&config.OutputFile, "o", "", "Output file path (shorthand).")
	fs.BoolVar(&config.Quiet, "quiet", false, "Quiet mode - minimal output for scripts.")
	fs.BoolVar(&config.Quiet, "q", false, "Quiet mode (shorthand).")
	fs.BoolVar(&config.HexOutput, "hex", false, "Display result in hexadecimal format.")
	fs.BoolVar(&config.Interactive, "interactive", false, "Start in interactive REPL mode.")
	fs.StringVar(&config.Completion, "completion", "", "Generate shell completion script (bash, zsh, fish, powershell).")
	fs.BoolVar(&config.Concise, "calculate", false, "Display the calculated value (disabled by default).")
	fs.BoolVar(&config.Concise, "c", false, "Display the calculated value (shorthand).")

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
