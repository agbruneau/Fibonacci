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
		return errors.New("la valeur du délai d’expiration (timeout) doit être strictement positive")
	}
	if c.Threshold < 0 {
		return fmt.Errorf("le seuil de parallélisation ne peut pas être négatif : %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return fmt.Errorf("le seuil FFT ne peut pas être négatif : %d", c.FFTThreshold)
	}
	isAlgoAvailable := false
	for _, a := range availableAlgos {
		if a == c.Algo {
			isAlgoAvailable = true
			break
		}
	}
	if c.Algo != "all" && !isAlgoAvailable {
		return fmt.Errorf("algorithme non reconnu : '%s'. Algorithmes valides : 'all' ou [%s]", c.Algo, strings.Join(availableAlgos, ", "))
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
	algoHelp := fmt.Sprintf("Algorithme à utiliser : 'all' (défaut) ou un des suivants [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 250000000, "Index n du nombre de Fibonacci à calculer.")
	fs.BoolVar(&config.Verbose, "v", false, "Afficher la valeur complète du résultat (peut être très long).")
	fs.BoolVar(&config.Details, "d", false, "Afficher les détails de performance et les métadonnées du résultat.")
	fs.BoolVar(&config.Details, "details", false, "Alias pour -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Durée maximale d’exécution du calcul.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", DefaultParallelThreshold, "Seuil (en bits) d’activation de la parallélisation des multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Seuil (en bits) pour activer la multiplication FFT (0 pour désactiver).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Exécute le mode calibration pour déterminer le seuil optimal de parallélisation.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}
	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Erreur de configuration :", err)
		fs.Usage()
		return AppConfig{}, errors.New("invalid configuration")
	}
	return config, nil
}
