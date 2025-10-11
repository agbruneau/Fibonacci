// @module(main)
// @author(Jules)
// @date(2023-10-27)
// @version(1.1)
//
// @description(Ce module est le point d'entrée et la racine de composition de l'application.)
// @pedagogical(Illustre la séparation des préoccupations, la gestion du cycle de vie, la concurrence structurée et l'injection de dépendances en Go.)
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
	"time"

	"golang.org/x/sync/errgroup"

	"example.com/fibcalc/internal/cli"
	"example.com/fibcalc/internal/fibonacci"
)

// @const(Codes de sortie standard pour la CLI.)
const (
	ExitSuccess       = 0   // Opération réussie.
	ExitErrorGeneric  = 1   // Erreur générique.
	ExitErrorTimeout  = 2   // Timeout atteint.
	ExitErrorMismatch = 3   // Incohérence des résultats.
	ExitErrorConfig   = 4   // Erreur de configuration.
	ExitErrorCanceled = 130 // Annulation par signal (Ctrl+C).
)

const (
	// @const(ProgressBufferMultiplier)
	// @description(Multiplicateur pour la taille du buffer du canal de progression.)
	// @rationale(Un buffer découple les producteurs des consommateurs, améliorant la performance en cas de lenteur de l'UI.)
	ProgressBufferMultiplier = 10
)

// @struct(AppConfig)
// @description(Agrège la configuration de l'application.)
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

// @method(Validate)
// @description(Valide la sémantique de la configuration.)
// @rationale("Fail-Fast" : Assure que les combinaisons de paramètres sont logiques avant l'exécution.)
func (c AppConfig) Validate(availableAlgos []string) error {
	if c.Timeout <= 0 {
		return errors.New("le timeout doit être positif")
	}
	if c.Threshold < 0 {
		return fmt.Errorf("le seuil de parallélisme ne peut être négatif : %d", c.Threshold)
	}
	if c.FFTThreshold < 0 {
		return fmt.Errorf("le seuil FFT ne peut être négatif : %d", c.FFTThreshold)
	}
	if c.Algo != "all" {
		if _, ok := calculatorRegistry[c.Algo]; !ok {
			return fmt.Errorf("algorithme inconnu : '%s'. Valides : 'all' ou [%s]", c.Algo, strings.Join(availableAlgos, ", "))
		}
	}
	return nil
}

// @registry(calculatorRegistry)
// @description(Registre des implémentations de `fibonacci.Calculator`.)
// @pattern(Registry)
// @rationale(Permet un couplage faible et respecte le Principe Ouvert/Fermé (SOLID).)
var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
}

// @function(init)
// @description(Initialisation du module, exécutée avant `main`.)
// @rationale(Utilisé pour une "vérification de santé" : prévient les panics dues à un registre mal configuré.)
func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Initialisation : le calculateur '%s' est nil.", name))
		}
	}
}

// @function(getSortedCalculatorKeys)
// @description(Retourne les clés du registre, triées alphabétiquement.)
// @rationale(Assure un affichage déterministe et cohérent à chaque exécution.)
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// @function(main)
// @description(Point d'entrée de l'application.)
// @architecture(Rôle : gestion des interactions avec l'OS et orchestration de haut niveau.)
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

// @function(parseConfig)
// @description(Analyse les arguments de la ligne de commande et retourne une configuration validée.)
// @pedagogical(La création d'un `flag.NewFlagSet` local et l'injection des dépendances (args, errorWriter) rendent la fonction pure et testable unitairement.)
func parseConfig(programName string, args []string, errorWriter io.Writer) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)

	availableAlgos := getSortedCalculatorKeys()
	algoHelp := fmt.Sprintf("Algorithme : 'all' ou l'un de [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 250000000, "Indice 'n' de la suite de Fibonacci.")
	fs.BoolVar(&config.Verbose, "v", false, "Affichage complet du résultat.")
	fs.BoolVar(&config.Details, "d", false, "Affichage des détails de performance.")
	fs.BoolVar(&config.Details, "details", false, "Affichage des détails de performance.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Délai d'exécution maximum.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", fibonacci.DefaultParallelThreshold, "Seuil (bits) pour le parallélisme.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Seuil (bits) pour la multiplication FFT (0 pour désactiver).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Mode calibration pour trouver le seuil de parallélisme optimal.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err
	}

	config.Algo = strings.ToLower(config.Algo)

	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Erreur:", err)
		fs.Usage()
		return AppConfig{}, errors.New("configuration invalide")
	}

	return config, nil
}

// @struct(CalculationResult)
// @description(DTO pour stocker le résultat et les métadonnées d'un calcul.)
type CalculationResult struct {
	Name     string
	Result   *big.Int
	Duration time.Duration
	Err      error
}

// @function(runCalibration)
// @description(Exécute une série de benchmarks pour déterminer le seuil de parallélisme optimal.)
func runCalibration(ctx context.Context, config AppConfig, out io.Writer) int {
	fmt.Fprintln(out, "--- Mode Calibration : Recherche du Seuil de Parallélisme Optimal ---")
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintln(out, "Erreur : Algorithme 'fast' non trouvé, requis pour la calibration.")
		return ExitErrorGeneric
	}

	thresholdsToTest := []int{0, 256, 512, 1024, 2048, 4096, 8192, 16384} // 0 = séquentiel

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
			fmt.Fprintln(out, "\nCalibration annulée.")
			return ExitErrorCanceled
		}
		thresholdLabel := fmt.Sprintf("%d bits", threshold)
		if threshold == 0 {
			thresholdLabel = "Séquentiel"
		}
		fmt.Fprintf(out, "Test du seuil : %-12s...", thresholdLabel)

		startTime := time.Now()
		_, err := calculator.Calculate(ctx, nil, 0, calibrationN, threshold, 0) // FFT désactivé
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

	// Affichage des résultats tabulés
	fmt.Fprintln(out, "\n--- Résultats de la Calibration ---")
	fmt.Fprintf(out, "  %-12s │ %s\n", "Seuil", "Durée")
	fmt.Fprintf(out, "  %s┼%s\n", strings.Repeat("─", 14), strings.Repeat("─", 20))
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
			highlight = " (Meilleur)"
		}
		fmt.Fprintf(out, "  %-12s │ %s%s\n", thresholdLabel, durationStr, highlight)
	}
	fmt.Fprintf(out, "\n✅ Recommandation: --threshold %d\n", bestThreshold)
	return ExitSuccess
}

// @function(run)
// @description(Contient la logique principale de l'application, rendue testable par l'injection de dépendances.)
// @architecture(Démontre la composition de contextes pour un arrêt propre (graceful shutdown).)
func run(ctx context.Context, config AppConfig, out io.Writer) int {
	if config.Calibrate {
		return runCalibration(ctx, config, out)
	}

	ctx, cancelTimeout := context.WithTimeout(ctx, config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	fmt.Fprintln(out, "--- Configuration ---")
	fmt.Fprintf(out, "Calcul de F(%d) avec un timeout de %s.\n", config.N, config.Timeout)
	fmt.Fprintf(out, "Système: %d CPU, Runtime: %s.\n", runtime.NumCPU(), runtime.Version())
	fmt.Fprintf(out, "Seuils: Parallèle=%d bits, FFT=%d bits.\n", config.Threshold, config.FFTThreshold)

	calculatorsToRun := getCalculatorsToRun(config)
	if len(calculatorsToRun) > 1 {
		fmt.Fprintln(out, "Mode: Comparaison parallèle.")
	} else {
		fmt.Fprintf(out, "Mode: Exécution simple (%s).\n", calculatorsToRun[0].Name())
	}
	fmt.Fprintln(out, "\n--- Exécution ---")

	results := executeCalculations(ctx, calculatorsToRun, config, out)

	if len(results) == 1 {
		res := results[0]
		fmt.Fprintln(out, "\n--- Résultat Final ---")
		if res.Err != nil {
			return handleCalculationError(res.Err, res.Duration, config.Timeout, out)
		}
		cli.DisplayResult(res.Result, config.N, res.Duration, config.Verbose, config.Details, out)
		return ExitSuccess
	}

	return analyzeComparisonResults(results, config, out)
}

// @function(getCalculatorsToRun)
// @description(Sélectionne les calculateurs à exécuter en fonction de la configuration.)
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

// @function(executeCalculations)
// @description(Orchestre l'exécution concurrente des calculs.)
// @pattern(Fan-Out / Fan-In)
// @architecture(Lance une goroutine par calcul (fan-out) et synchronise leur achèvement (fan-in).)
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, config AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	// Fan-Out : Lancement des workers
	for i, calc := range calculators {
		idx, calculator := i, calc // Capture de variable pour la goroutine
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, config.N, config.Threshold, config.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			// Un échec ne doit pas annuler les autres calculs dans un benchmark.
			return nil
		})
	}

	// Goroutine pour l'affichage de la progression
	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayAggregateProgress(&displayWg, progressChan, len(calculators), out)

	// Fan-In : Synchronisation
	_ = g.Wait()       // Attendre la fin des calculs
	close(progressChan) // Fermer le canal pour signaler la fin à l'UI
	displayWg.Wait()    // Attendre la fin de l'affichage

	return results
}

// @function(analyzeComparisonResults)
// @description(Analyse et affiche les résultats du mode de comparaison.)
func analyzeComparisonResults(results []CalculationResult, config AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil // Succès d'abord
		}
		return results[i].Duration < results[j].Duration // Puis par durée
	})

	var firstValidResult *big.Int
	var firstError error
	successCount := 0

	// Préparation pour affichage tabulaire
	col1Width, col2Width, col3Width := len("Algorithme"), len("Durée"), len("Statut")
	type displayRow struct{ Name, Duration, Status string }
	displayData := make([]displayRow, len(results))

	for i, res := range results {
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
		displayData[i] = displayRow{res.Name, res.Duration.String(), status}
		if len(res.Name) > col1Width {
			col1Width = len(res.Name)
		}
		if len(res.Duration.String()) > col2Width {
			col2Width = len(res.Duration.String())
		}
		if len(status) > col3Width {
			col3Width = len(status)
		}
	}

	// Affichage
	fmt.Fprintln(out, "\n--- Résultats de la Comparaison ---")
	rowFormat := fmt.Sprintf("  %%-%ds │ %%-%ds │ %%s\n", col1Width, col2Width)
	fmt.Fprintf(out, rowFormat, "Algorithme", "Durée", "Statut")
	separator := fmt.Sprintf("  %s┼%s┼%s", strings.Repeat("─", col1Width+1), strings.Repeat("─", col2Width+2), strings.Repeat("─", col3Width+2))
	fmt.Fprintln(out, separator)
	for _, data := range displayData {
		fmt.Fprintf(out, rowFormat, data.Name, data.Duration, data.Status)
	}

	if successCount == 0 {
		fmt.Fprintln(out, "\nStatut Global: Échec. Aucun calcul n'a réussi.")
		return handleCalculationError(firstError, 0, config.Timeout, out)
	}

	// Validation croisée
	mismatch := false
	for _, res := range results {
		if res.Err == nil && res.Result.Cmp(firstValidResult) != 0 {
			mismatch = true
			break
		}
	}
	if mismatch {
		fmt.Fprintln(out, "\nStatut Global: Échec Critique! Incohérence des résultats.")
		return ExitErrorMismatch
	}

	fmt.Fprintln(out, "\nStatut Global: Succès. Tous les résultats valides sont identiques.")
	cli.DisplayResult(firstValidResult, config.N, 0, config.Verbose, config.Details, out)
	return ExitSuccess
}

// @function(handleCalculationError)
// @description(Interprète une erreur et retourne le code de sortie approprié.)
// @pedagogical(L'utilisation de `errors.Is` est plus robuste que `==` car elle permet de "déballer" les erreurs enveloppées.)
func handleCalculationError(err error, duration time.Duration, timeout time.Duration, out io.Writer) int {
	if err == nil {
		return ExitSuccess
	}
	msgSuffix := ""
	if duration > 0 {
		msgSuffix = fmt.Sprintf(" après %s", duration)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintf(out, "Statut: Échec (Timeout). Le délai de %s a été dépassé%s.\n", timeout, msgSuffix)
		return ExitErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintf(out, "Statut: Annulé%s.\n", msgSuffix)
		return ExitErrorCanceled
	}
	fmt.Fprintf(out, "Statut: Échec. Erreur inattendue: %v\n", err)
	return ExitErrorGeneric
}
