// Le paquetage main est le point d'entrée de l'application fibcalc. Il gère
// l'analyse des arguments de la ligne de commande, la configuration,
// l'orchestration des calculs et l'affichage des résultats.
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
	"example.com/fibcalc/internal/fibonacci"
)

// Codes de sortie de l'application.
const (
	ExitSuccess       = 0
	ExitErrorGeneric  = 1
	ExitErrorTimeout  = 2
	ExitErrorMismatch = 3
	ExitErrorConfig   = 4
	ExitErrorCanceled = 130
)

const ProgressBufferMultiplier = 10

// AppConfig agrège les paramètres de configuration de l'application.
type AppConfig struct {
	N            uint64
	Verbose      bool
	Details      bool
	Timeout      time.Duration
	Algo         string
	Threshold    int
	FFTThreshold int
	Calibrate    bool
}

// Validate vérifie la cohérence sémantique de la configuration.
func (c AppConfig) Validate(availableAlgos []string) error {
	if c.Timeout <= 0 {
		return errors.New("la valeur du timeout doit être strictement positive")
	}
	if c.Threshold < 0 {
		return fmt.Errorf("le seuil de parallélisme ne peut être négatif : %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return fmt.Errorf("le seuil FFT ne peut être négatif : %d", c.FFTThreshold)
	}
	if c.Algo != "all" {
		if _, ok := calculatorRegistry[c.Algo]; !ok {
			return fmt.Errorf("algorithme non reconnu : '%s'. Algorithmes valides : 'all' ou l'un de [%s]", c.Algo, strings.Join(availableAlgos, ", "))
		}
	}
	return nil
}

var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
	"fft":    fibonacci.NewCalculator(&fibonacci.FFTBasedCalculator{}),
}

func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Erreur critique d'initialisation : le calculateur enregistré sous le nom '%s' est nil.", name))
		}
	}
}

// getSortedCalculatorKeys retourne les clés du registre des calculateurs, triées.
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func main() {
	config, err := parseConfig(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(ExitSuccess)
		}
		os.Exit(ExitErrorConfig)
	}
	exitCode := run(context.Background(), config, os.Stdout)
	os.Exit(exitCode)
}

// parseConfig analyse les arguments de la ligne de commande.
func parseConfig(programName string, args []string, errorWriter io.Writer) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)
	availableAlgos := getSortedCalculatorKeys()
	algoHelp := fmt.Sprintf("Algorithme à utiliser : 'all' (défaut) ou l'un de [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 250000000, "Indice 'n' du nombre de Fibonacci à calculer.")
	fs.BoolVar(&config.Verbose, "v", false, "Afficher la valeur complète du résultat (peut être très long).")
	fs.BoolVar(&config.Details, "d", false, "Afficher les détails de performance et les métadonnées du résultat.")
	fs.BoolVar(&config.Details, "details", false, "Alias pour -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Délai d'exécution maximal pour le calcul.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", fibonacci.DefaultParallelThreshold, "Seuil (en nombre de bits) pour activer la parallélisation des multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Seuil (en nombre de bits) pour utiliser la multiplication par FFT (0 pour désactiver).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Exécuter le mode de calibration pour déterminer le seuil de parallélisme optimal.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}
	config.Algo = strings.ToLower(config.Algo)
	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Erreur de configuration :", err)
		fs.Usage()
		return AppConfig{}, errors.New("configuration invalide")
	}
	return config, nil
}

// CalculationResult encapsule le résultat d'un calcul.
type CalculationResult struct {
	Name     string
	Result   *big.Int
	Duration time.Duration
	Err      error
}

// runCalibration exécute des benchmarks pour trouver le seuil de parallélisme optimal.
func runCalibration(ctx context.Context, config AppConfig, out io.Writer) int {
	fmt.Fprintln(out, "--- Mode Calibration : Recherche du Seuil de Parallélisme Optimal ---")
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintln(out, "Erreur critique : L'algorithme 'fast' est requis pour la calibration mais n'a pas été trouvé.")
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

	for _, threshold := range thresholdsToTest {
		if ctx.Err() != nil {
			fmt.Fprintln(out, "\nCalibration interrompue.")
			return ExitErrorCanceled
		}
		thresholdLabel := fmt.Sprintf("%d bits", threshold)
		if threshold == 0 {
			thresholdLabel = "Séquentiel"
		}
		fmt.Fprintf(out, "Test du seuil : %-12s...", thresholdLabel)
		startTime := time.Now()
		_, err := calculator.Calculate(ctx, nil, 0, calibrationN, threshold, 0)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Fprintf(out, " ❌ Échec (%v)\n", err)
			results = append(results, calibrationResult{threshold, 0, err})
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return handleCalculationError(err, duration, config.Timeout, out)
			}
			continue
		}

		fmt.Fprintf(out, " ✅ Succès (Durée: %s)\n", duration)
		results = append(results, calibrationResult{threshold, duration, nil})
		if duration < bestDuration {
			bestDuration, bestThreshold = duration, threshold
		}
	}

	fmt.Fprintln(out, "\n--- Synthèse de la Calibration ---")
	fmt.Fprintf(out, "  %-12s │ %s\n", "Seuil", "Durée d'exécution")
	fmt.Fprintf(out, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 25))
	for _, res := range results {
		thresholdLabel := fmt.Sprintf("%d bits", res.Threshold)
		if res.Threshold == 0 {
			thresholdLabel = "Séquentiel"
		}
		durationStr := "N/A"
		if res.Err == nil {
			durationStr = res.Duration.String()
		}
		highlight := ""
		if res.Threshold == bestThreshold && res.Err == nil {
			highlight = " (Optimal)"
		}
		fmt.Fprintf(out, "  %-12s │ %s%s\n", thresholdLabel, durationStr, highlight)
	}
	fmt.Fprintf(out, "\n✅ Recommandation pour cette machine : --threshold %d\n", bestThreshold)
	return ExitSuccess
}

// run est la fonction principale qui orchestre l'exécution de l'application.
func run(ctx context.Context, config AppConfig, out io.Writer) int {
	if config.Calibrate {
		return runCalibration(ctx, config, out)
	}
	ctx, cancelTimeout := context.WithTimeout(ctx, config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	fmt.Fprintln(out, "--- Configuration d'Exécution ---")
	fmt.Fprintf(out, "Calcul de F(%d) avec un timeout de %s.\n", config.N, config.Timeout)
	fmt.Fprintf(out, "Environnement : %d CPU logiques, Go %s.\n", runtime.NumCPU(), runtime.Version())
	fmt.Fprintf(out, "Seuils d'optimisation : Parallélisme=%d bits, FFT=%d bits.\n", config.Threshold, config.FFTThreshold)

	calculatorsToRun := getCalculatorsToRun(config)
	if len(calculatorsToRun) > 1 {
		fmt.Fprintln(out, "Mode d'exécution : Comparaison parallèle de tous les algorithmes.")
	} else {
		fmt.Fprintf(out, "Mode d'exécution : Calcul simple avec l'algorithme %s.\n", calculatorsToRun[0].Name())
	}
	fmt.Fprintln(out, "\n--- Début de l'Exécution ---")

	results := executeCalculations(ctx, calculatorsToRun, config, out)
	return analyzeComparisonResults(results, config, out)
}

// getCalculatorsToRun sélectionne les calculateurs à exécuter.
func getCalculatorsToRun(config AppConfig) []fibonacci.Calculator {
	if config.Algo == "all" {
		keys := getSortedCalculatorKeys()
		calculators := make([]fibonacci.Calculator, len(keys))
		for i, k := range keys {
			calculators[i] = calculatorRegistry[k]
		}
		return calculators
	}
	return []fibonacci.Calculator{calculatorRegistry[config.Algo]}
}

// executeCalculations orchestre l'exécution concurrente des calculs.
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, config AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	for i, calc := range calculators {
		idx, calculator := i, calc
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, config.N, config.Threshold, config.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			return nil
		})
	}

	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayAggregateProgress(&displayWg, progressChan, len(calculators), out)

	_ = g.Wait()
	close(progressChan)
	displayWg.Wait()

	return results
}

// analyzeComparisonResults analyse et affiche les résultats.
func analyzeComparisonResults(results []CalculationResult, config AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil
		}
		return results[i].Duration < results[j].Duration
	})

	var firstValidResult *big.Int
	var firstError error
	successCount := 0

	fmt.Fprintln(out, "\n--- Synthèse de la Comparaison ---")
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "Algorithme\tDurée\tStatut")
	fmt.Fprintln(tw, "----------\t-----\t------")
	for _, res := range results {
		var status string
		if res.Err != nil {
			status = fmt.Sprintf("❌ Échec (%v)", res.Err)
			if firstError == nil {
				firstError = res.Err
			}
		} else {
			status = "✅ Succès"
			successCount++
			if firstValidResult == nil {
				firstValidResult = res.Result
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", res.Name, res.Duration.String(), status)
	}
	tw.Flush()

	if successCount == 0 {
		fmt.Fprintln(out, "\nStatut Global : Échec. Aucun des algorithmes n'a pu terminer le calcul.")
		return handleCalculationError(firstError, 0, config.Timeout, out)
	}

	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		fmt.Fprintln(out, "\nStatut Global : ÉCHEC CRITIQUE ! Une incohérence a été détectée entre les résultats des algorithmes.")
		return ExitErrorMismatch
	}

	fmt.Fprintln(out, "\nStatut Global : Succès. Tous les résultats valides sont cohérents.")
	cli.DisplayResult(firstValidResult, config.N, 0, config.Verbose, config.Details, out)
	return ExitSuccess
}

// handleCalculationError interprète une erreur et retourne le code de sortie approprié.
func handleCalculationError(err error, duration time.Duration, timeout time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" après %s", duration)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "Statut : Échec (Timeout). Le délai d'exécution de %s a été dépassé%s.\n", timeout, msgSuffix)
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "Statut : Annulé par l'utilisateur%s.\n", msgSuffix)
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "Statut : Échec. Une erreur inattendue est survenue : %v\n", err)
	return ExitErrorGeneric
}