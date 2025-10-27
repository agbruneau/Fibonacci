// Package config provides the configuration management for the fibcalc application.
// It defines the data structure for the configuration, handles the parsing of
// command-line arguments, and performs validation on the configuration values.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	// DefaultParallelThreshold defines the bit threshold from which
	// multiplications of large integers are parallelized.
	DefaultParallelThreshold = 4096
)

// AppConfig aggregates the application's configuration parameters, parsed from
// command-line flags. It encapsulates all settings that control the execution,
// from the Fibonacci index to calculate to performance tuning parameters.
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
	// Calibrate, if true, runs the application in calibration mode to find the
	// optimal parallelism threshold.
	Calibrate bool
}

// Validate checks the semantic consistency of the configuration parameters. It
// ensures that numerical values are within valid ranges and that the chosen
// algorithm is supported.
//
// Parameters:
//   - availableAlgos: A slice of the names of the available algorithms to
//     validate the `Algo` field against.
//
// Returns an error if the configuration is invalid, otherwise nil.
func (c AppConfig) Validate(availableAlgos []string) error {
	if c.Timeout <= 0 {
		return errors.New("timeout value must be strictly positive")
	}
	if c.Threshold < 0 {
		return fmt.Errorf("parallelism threshold cannot be negative: %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return fmt.Errorf("FFT threshold cannot be negative: %d", c.FFTThreshold)
	}
	isAlgoAvailable := false
	for _, a := range availableAlgos {
		if a == c.Algo {
			isAlgoAvailable = true
			break
		}
	}
	if c.Algo != "all" && !isAlgoAvailable {
		return fmt.Errorf("unrecognized algorithm: '%s'. Valid algorithms: 'all' or one of [%s]", c.Algo, strings.Join(availableAlgos, ", "))
	}
	return nil
}

// ParseConfig parses the command-line arguments and populates an `AppConfig`
// struct. It defines all the command-line flags, sets their default values, and
// handles the parsing process. After parsing, it performs validation on the
// resulting configuration.
//
// The function is designed to be testable by allowing the input arguments and
// output writer to be specified.
//
// Parameters:
//   - programName: The name of the program, used for the help message.
//   - args: A slice of strings representing the command-line arguments.
//   - errorWriter: The `io.Writer` to which parsing errors will be written.
//   - availableAlgos: A slice of the names of the available algorithms.
//
// Returns an `AppConfig` struct and an error if parsing or validation fails.
func ParseConfig(programName string, args []string, errorWriter io.Writer, availableAlgos []string) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)
	algoHelp := fmt.Sprintf("Algorithm to use: 'all' (default) or one of [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 250000000, "Index 'n' of the Fibonacci number to calculate.")
	fs.BoolVar(&config.Verbose, "v", false, "Display the full value of the result (can be very long).")
	fs.BoolVar(&config.Details, "d", false, "Display performance details and result metadata.")
	fs.BoolVar(&config.Details, "details", false, "Alias for -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Maximum execution time for the calculation.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", DefaultParallelThreshold, "Threshold (in bits) to enable parallelization of multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Threshold (in bits) to use FFT multiplication (0 to disable).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Run calibration mode to determine the optimal parallelism threshold.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}
	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Configuration error:", err)
		fs.Usage()
		return AppConfig{}, errors.New("invalid configuration")
	}
	return config, nil
}
