// The main package is the entry point of the fibcalc application. It handles
// command-line argument parsing, configuration, calculation orchestration,
// and result display.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"golang.org/x/sync/errgroup"

	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/config"
	"example.com/fibcalc/internal/fibonacci"
	"example.com/fibcalc/internal/i18n"
)

// Application exit codes define the standard exit statuses for the application.
const (
	// ExitSuccess indicates a successful execution without errors.
	ExitSuccess = 0
	// ExitErrorGeneric indicates a general, unspecified error.
	ExitErrorGeneric = 1
	// ExitErrorTimeout signals that the calculation exceeded the configured timeout.
	ExitErrorTimeout = 2
	// ExitErrorMismatch indicates an inconsistency detected between the results of different algorithms.
	ExitErrorMismatch = 3
	// ExitErrorConfig denotes an error related to configuration or command-line arguments.
	ExitErrorConfig = 4
	// ExitErrorCanceled is used when the execution is canceled by the user (e.g., via SIGINT).
	ExitErrorCanceled = 130
)

const (
	// ANSI escape codes for text styling.
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorGreen     = "\033[32m"
	ColorYellow    = "\033[33m"
	ColorBlue      = "\033[34m"
	ColorMagenta   = "\033[35m"
	ColorCyan      = "\033[36m"
	ColorBold      = "\033[1m"
	ColorUnderline = "\033[4m"
)

// ProgressBufferMultiplier defines the buffer size of the progress channel,
// calculated as a multiple of the number of active calculators. A larger
// buffer reduces the risk of blocking progress updates.
const ProgressBufferMultiplier = 10

var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
	"fft":    fibonacci.NewCalculator(&fibonacci.FFTBasedCalculator{}),
}

func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Erreur d'initialisation critique : le calculateur enregistré sous le nom '%s' est nul.", name))
		}
	}
}

// getSortedCalculatorKeys returns the sorted keys of the calculator registry.
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// main is the entry point of the application. It parses the command-line
// arguments, validates the configuration, and orchestrates the execution of the
// Fibonacci calculation. The application's exit code is determined by the
// outcome of the `run` function.
func main() {
	availableAlgos := getSortedCalculatorKeys()
	cfg, err := config.ParseConfig(os.Args[0], os.Args[1:], os.Stderr, availableAlgos)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(ExitSuccess)
		}
		os.Exit(ExitErrorConfig)
	}
	// Chargement i18n optionnel
	if cfg.I18nDir != "" {
		if err := i18n.LoadFromDir(cfg.I18nDir, cfg.Lang); err != nil {
			// Non bloquant : on continue avec les messages intégrés
			fmt.Fprintln(os.Stderr, "[i18n] chargement des traductions échoué:", err)
		}
	}
	exitCode := run(context.Background(), cfg, os.Stdout)
	os.Exit(exitCode)
}

// CalculationResult encapsulates the outcome of a single Fibonacci calculation.
// It holds the result, execution duration, and any error that occurred, facilitating
// the aggregation and comparison of results from multiple algorithms.
//
// Fields:
//   - Name: The identifier of the algorithm used for the calculation.
//   - Result: The calculated Fibonacci number. It is nil if an error occurred.
//   - Duration: The total time taken for the calculation.
//   - Err: Any error encountered during the calculation.
type CalculationResult struct {
	// Name is the identifier of the algorithm used for the calculation.
	Name string
	// Result is the calculated Fibonacci number. It is nil if an error occurred.
	Result *big.Int
	// Duration is the total time taken for the calculation.
	Duration time.Duration
	// Err holds any error encountered during the calculation.
	Err error
}

// runCalibration runs a series of benchmarks to determine the optimal parallelism
// threshold for the current machine. It tests a predefined set of threshold
// values and measures the execution time for each, ultimately recommending the
// value that yields the best performance. This function is invoked when the
// `--calibrate` flag is provided.
//
// The calibration process involves:
// - Iterating through a list of threshold values.
// - For each threshold, calculating a large Fibonacci number.
// - Recording the duration of each calculation.
// - Displaying a summary table comparing the performance of each threshold.
// - Recommending the threshold that resulted in the shortest execution time.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - cfg: The application's configuration, used for settings like timeout.
//   - out: The output writer for displaying progress and results.
//
// Returns an exit code indicating the outcome of the calibration process.
func runCalibration(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	writeOut(out, "%s\n", i18n.Messages["CalibrationTitle"])
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		writeOut(out, "%sErreur critique : l'algorithme 'fast' est requis pour la calibration mais est introuvable.%s\n", ColorRed, ColorReset)
		return ExitErrorGeneric
	}

	thresholdsToTest := []int{0, 256, 512, 1024, 2048, 4096, 8192, 16384}
	type calibrationResult struct {
		Threshold int
		Duration  time.Duration
		Err       error
	}
	results := make([]calibrationResult, 0, len(thresholdsToTest))
	bestDuration := time.Duration(1<<63 - 1)
	bestThreshold := 0

	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 1*ProgressBufferMultiplier)
	wg.Add(1)
	go cli.DisplayProgress(&wg, progressChan, 1, out)

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			writeOut(out, "\n%sCalibration interrompue.%s\n", ColorYellow, ColorReset)
			return ExitErrorCanceled
		}

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, progressChan, 0, calibrationN, threshold, 0)
		duration := time.Since(startTime)

		if err != nil {
			writeOut(out, "%s❌ Échec (%v)%s\n", ColorRed, err, ColorReset)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				close(progressChan)
				wg.Wait()
				return handleCalculationError(err, duration, cfg.Timeout, out)
			}
			continue
		}

		results = append(results, calibrationResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}
	close(progressChan)
	wg.Wait()

	// Raffinement local (recherche type dichotomique autour du meilleur seuil)
	// Hypothèse raisonnable: la courbe temps(seuil) est localement régulière.
	if bestDuration > 0 {
		maxBound := 65536
		step := bestThreshold / 4
		if step < 128 {
			step = 128
		}
		refineCtx, cancel := context.WithTimeout(ctx, cfg.Timeout/4)
		defer cancel()
		for iter := 0; iter < 5 && step >= 64; iter++ {
			candidates := []int{bestThreshold - step, bestThreshold + step}
			for _, cand := range candidates {
				if cand < 0 {
					cand = 0
				}
				if cand > maxBound {
					cand = maxBound
				}
				startTime := time.Now()
				_, err := calculator.Calculate(refineCtx, nil, 0, calibrationN, cand, 0)
				duration := time.Since(startTime)
				results = append(results, calibrationResult{cand, duration, err})
				if err == nil && duration < bestDuration {
					bestDuration, bestThreshold = duration, cand
				}
			}
			step /= 2
		}
	}

	writeOut(out, "\n%s\n", i18n.Messages["CalibrationSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	writeOut(tw, "  %sSeuil%s        │ %sTemps d'exécution%s\n", ColorUnderline, ColorReset, ColorUnderline, ColorReset)
	writeOut(tw, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Séquentiel"
		}
		durationStr := fmt.Sprintf("%sN/A%s", ColorRed, ColorReset)
		if res.Err == nil {
			durationStr = cli.FormatExecutionDuration(res.Duration)
			if res.Duration == 0 {
				durationStr = "< 1µs"
			}
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = fmt.Sprintf(" %s(Optimal)%s", ColorGreen, ColorReset)
		}
		writeOut(tw, "  %s%-12s%s │ %s%s%s%s\n", ColorCyan, thresholdLabel, ColorReset, ColorYellow, durationStr, ColorReset, highlight)
	}
	tw.Flush()
	writeOut(out, "\n%s✅ Recommandation pour cette machine : %s--threshold %d%s\n",
		ColorGreen, ColorYellow, bestThreshold, ColorReset)
	return ExitSuccess
}

// run is the main function that orchestrates the application's execution flow.
// It is responsible for setting up the execution context, including timeouts and
// signal handling, and then initiating the Fibonacci calculations.
//
// The process includes:
// - Configuring a context for timeout and graceful shutdown.
// - Displaying the execution configuration to the user.
// - Selecting the appropriate calculator(s) based on the configuration.
// - Executing the calculation(s).
// - Analyzing and displaying the results.
//
// Parameters:
//   - ctx: The parent context for the execution.
//   - cfg: The application's configuration.
//   - out: The output writer for displaying information and results.
//
// Returns an exit code that reflects the outcome of the execution.
func run(ctx context.Context, cfg config.AppConfig, out io.Writer) int {
	if cfg.Calibrate {
		return runCalibration(ctx, cfg, out)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, cfg.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Calibration automatique rapide au démarrage (si activée)
	if cfg.AutoCalibrate {
		if updated, ok := autoCalibrate(ctx, cfg, out); ok {
			cfg = updated
		}
	}

	writeOut(out, "%s\n", i18n.Messages["ExecConfigTitle"])
	writeOut(out, "Calcul de %sF(%d)%s avec un délai maximum de %s%s%s.\n",
		ColorMagenta, cfg.N, ColorReset, ColorYellow, cfg.Timeout, ColorReset)
	writeOut(out, "Environnement : %s%d%s processeurs logiques, Go %s%s%s.\n",
		ColorCyan, runtime.NumCPU(), ColorReset, ColorCyan, runtime.Version(), ColorReset)
	writeOut(out, "Seuils d'optimisation : Parallélisme=%s%d%s bits, FFT=%s%d%s bits.\n",
		ColorCyan, cfg.Threshold, ColorReset, ColorCyan, cfg.FFTThreshold, ColorReset)

	calculatorsToRun := getCalculatorsToRun(cfg)
	var modeDesc string
	if len(calculatorsToRun) > 1 {
		modeDesc = "Comparaison parallèle de tous les algorithmes"
	} else {
		modeDesc = fmt.Sprintf("Calcul simple avec l'algorithme %s%s%s",
			ColorGreen, calculatorsToRun[0].Name(), ColorReset)
	}
	writeOut(out, "Mode d'exécution : %s.\n", modeDesc)
	writeOut(out, "\n%s\n", i18n.Messages["ExecStartTitle"])

	results := executeCalculations(ctx, calculatorsToRun, cfg, out)
	return analyzeComparisonResults(results, cfg, out)
}

// autoCalibrate effectue une calibration rapide des seuils de parallélisation
// et du seuil FFT pour la machine courante. Elle est courte et opportuniste :
// si le contexte est annulé ou si un essai dépasse une petite fraction du
// timeout, on conserve les valeurs actuelles.
// Retourne (cfgMisAJour, true) si mise à jour, sinon (cfgOriginal, false).
func autoCalibrate(parentCtx context.Context, cfg config.AppConfig, out io.Writer) (config.AppConfig, bool) {
	// Ne lance pas l'auto-calibration en mode comparaison de tous les algos :
	// on cible l'implémentation fast (doubling) pour vitesse et cohérence.
	calc := calculatorRegistry["fast"]
	if calc == nil {
		return cfg, false
	}

	// Fenêtre courte : chaque essai dispose d'au plus 1/6 du timeout global,
	// avec une borne inférieure utile pour éviter trop court (ex: 2s).
	perTrial := cfg.Timeout / 6
	if perTrial < 2*time.Second {
		perTrial = 2 * time.Second
	}

	// Taille d'entrée pour calibration: suffisamment grande pour déclencher
	// les chemins d'intérêt sans être trop longue.
	const nForCalibration = 10_000_000

	tryRun := func(threshold, fftThreshold int) (time.Duration, error) {
		ctx, cancel := context.WithTimeout(parentCtx, perTrial)
		defer cancel()
		start := time.Now()
		_, err := calc.Calculate(ctx, nil, 0, nForCalibration, threshold, fftThreshold)
		return time.Since(start), err
	}

	// 1) Calibration du seuil de parallélisme (FFT désactivée pour stabilité)
	parallelCandidates := []int{0, 512, 1024, 2048, 4096, 8192, 12288, 16384, 24576, 32768}
	bestPar := cfg.Threshold
	bestParDur := time.Duration(1<<63 - 1)
	for _, cand := range parallelCandidates {
		dur, err := tryRun(cand, 0)
		if err != nil {
			continue
		}
		if dur < bestParDur {
			bestParDur, bestPar = dur, cand
		}
	}

	// 2) Calibration du seuil FFT (en utilisant le meilleur parallélisme trouvé)
	fftCandidates := []int{0, 12000, 16000, 20000, 24000, 28000, 32000, 40000}
	bestFFT := cfg.FFTThreshold
	bestFFTDur := time.Duration(1<<63 - 1)
	for _, cand := range fftCandidates {
		dur, err := tryRun(bestPar, cand)
		if err != nil {
			continue
		}
		if dur < bestFFTDur {
			bestFFTDur, bestFFT = dur, cand
		}
	}

	// Si aucune mesure valide n'a été faite, ne rien changer
	if bestParDur == time.Duration(1<<63-1) && bestFFTDur == time.Duration(1<<63-1) {
		return cfg, false
	}

	// Appliquer les meilleures valeurs trouvées
	updated := cfg
	if bestParDur != time.Duration(1<<63-1) {
		updated.Threshold = bestPar
	}
	if bestFFTDur != time.Duration(1<<63-1) {
		updated.FFTThreshold = bestFFT
	}

	// Affichage succinct
	writeOut(out, "%sCalibration auto%s: parallélisme=%s%d%s bits, FFT=%s%d%s bits\n",
		ColorGreen, ColorReset, ColorYellow, updated.Threshold, ColorReset, ColorYellow, updated.FFTThreshold, ColorReset)
	return updated, true
}

// getCalculatorsToRun selects the calculators to be run based on the application's
// configuration. If the "all" algorithm is specified, it returns a list of all
// registered calculators. Otherwise, it returns the specific calculator that was
// requested.
//
// Parameters:
//   - cfg: The application's configuration, which specifies the desired algorithm.
//
// Returns a slice of `fibonacci.Calculator` instances to be executed.
func getCalculatorsToRun(cfg config.AppConfig) []fibonacci.Calculator {
	if cfg.Algo == "all" {
		keys := getSortedCalculatorKeys()
		calculators := make([]fibonacci.Calculator, len(keys))
		for i, k := range keys {
			calculators[i] = calculatorRegistry[k]
		}
		return calculators
	}
	return []fibonacci.Calculator{calculatorRegistry[cfg.Algo]}
}

// executeCalculations orchestrates the concurrent execution of one or more
// Fibonacci calculations. It uses an `errgroup` to manage the lifecycle of the
// calculation goroutines and ensures that they can be gracefully canceled.
//
// This function is responsible for:
// - Setting up a progress channel for real-time updates.
// - Launching a separate goroutine for the progress display.
// - Starting a goroutine for each calculation.
// - Aggregating the results of each calculation.
// - Waiting for all calculations and the progress display to complete.
//
// Parameters:
//   - ctx: The context for managing cancellation.
//   - calculators: A slice of `fibonacci.Calculator` instances to be executed.
//   - cfg: The application's configuration.
//   - out: The output writer for the progress display.
//
// Returns a slice of `CalculationResult`, with each element corresponding to the
// outcome of a single calculation.
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, cfg config.AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayProgress(&displayWg, progressChan, len(calculators), out)

	for i, calc := range calculators {
		idx, calculator := i, calc
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, cfg.N, cfg.Threshold, cfg.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			// We return nil because we want all calculators to complete, even if one fails.
			// The error is captured in the results slice and handled later.
			return nil
		})
	}

	g.Wait()
	close(progressChan)
	displayWg.Wait()

	return results
}

// analyzeComparisonResults processes and displays the final results of the
// calculations. It sorts the results by performance, checks for inconsistencies,
// and presents a summary to the user.
//
// The analysis includes the following steps:
// - Sorting the results by duration, with successful calculations prioritized.
// - Displaying a detailed comparison summary in a tabular format.
// - Checking for mismatches between the results of different algorithms.
// - Reporting the final status (success, failure, or mismatch).
// - Displaying the final calculated value and performance details.
//
// Parameters:
//   - results: The slice of `CalculationResult` from the calculations.
//   - cfg: The application's configuration.
//   - out: The output writer for displaying the analysis.
//
// Returns an exit code that reflects the outcome of the analysis.
func analyzeComparisonResults(results []CalculationResult, cfg config.AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil
		}
		return results[i].Duration < results[j].Duration
	})

	var firstValidResult *big.Int
	var firstValidResultDuration time.Duration
	var firstError error
	successCount := 0

	writeOut(out, "\n%s\n", i18n.Messages["ComparisonSummary"])
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	writeOut(tw, "%sAlgorithme%s\t%sDurée%s\t%sStatut%s\n",
		ColorUnderline, ColorReset, ColorUnderline, ColorReset, ColorUnderline, ColorReset)

	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("%s❌ Échec (%v)%s", ColorRed, res.Err, ColorReset)
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = fmt.Sprintf("%s✅ Succès%s", ColorGreen, ColorReset)
			successCount++
			if firstValidResult == nil {
				firstValidResult = res.Result
				firstValidResultDuration = res.Duration
			}
		}
		duree := cli.FormatExecutionDuration(res.Duration)
		if res.Duration == 0 {
			duree = "< 1µs"
		}
		writeOut(tw, "%s%s%s\t%s%s%s\t%s\n",
			ColorBlue, res.Name, ColorReset,
			ColorYellow, duree, ColorReset,
			status)
	}
	tw.Flush()

	if successCount == 0 {
		writeOut(out, "\n%s\n", i18n.Messages["GlobalStatusFailure"])
		return handleCalculationError(firstError, 0, cfg.Timeout, out)
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		writeOut(out, "\n"+i18n.Messages["StatusCriticalMismatch"])
		return ExitErrorMismatch
	}

	writeOut(out, "\n"+i18n.Messages["GlobalStatusSuccess"])
	cli.DisplayResult(firstValidResult, cfg.N, firstValidResultDuration, cfg.Verbose, cfg.Details, out)
	return ExitSuccess
}

// handleCalculationError interprets a calculation error and translates it into a
// human-readable message and a corresponding exit code. It handles specific
// error types, such as context cancellation and deadline exceeded, to provide
// more informative feedback to the user.
//
// Parameters:
//   - err: The error to be handled. If nil, the function returns ExitSuccess.
//   - duration: The execution duration at the time of the error.
//   - timeout: The configured timeout for the operation.
//   - out: The output writer for displaying the error message.
//
// Returns an exit code that corresponds to the nature of the error.
func handleCalculationError(err error, duration time.Duration, timeout time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" after %s%s%s", ColorYellow, duration, ColorReset)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		writeOut(out, "%s\n", i18n.Messages["StatusTimeout"])
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		writeOut(out, "%s%s%s.%s\n", ColorYellow, i18n.Messages["StatusCanceled"], msgSuffix, ColorReset)
		return ExitErrorCanceled
	}
	writeOut(out, "%s\n", i18n.Messages["StatusFailure"])
	return ExitErrorGeneric
}

// writeOut centralise l'écriture sur out et gère (ou loggue) l’erreur.
func writeOut(out io.Writer, format string, a ...interface{}) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		// Erreur d'I/O sur la sortie utilisateur, généralement critique !
		// Ici nous logguons sur stderr via fmt.Fprintln mais on pourrait exit immédiatement.
		fmt.Fprintln(os.Stderr, "[Erreur sortie] :", err)
		// os.Exit(1) // En production, on pourrait envisager un exit.
	}
}

// Messages centralisés : voir internal/i18n/messages.go
