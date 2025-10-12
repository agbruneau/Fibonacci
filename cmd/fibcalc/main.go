// @module(main)
// @author(Jules)
// @date(2023-10-27)
// @version(1.2)
//
// @description(Ce module constitue le point d'entrée de l'application et sa racine de composition ("composition root"). Il est responsable de l'initialisation, de la configuration, de l'orchestration des modules internes et de la gestion du cycle de vie du processus.)
// @pedagogical(Ce code sert d'illustration à plusieurs principes fondamentaux de l'ingénierie logicielle en Go : la séparation des préoccupations (le `main` ne contient aucune logique métier), la gestion du cycle de vie via la composition de contextes (`context`), la concurrence structurée avec `errgroup`, et l'injection de dépendances pour assurer une testabilité maximale.)
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

// @const(Codes de sortie standards définissant le protocole de communication avec le shell appelant.)
const (
	ExitSuccess       = 0   // L'opération s'est terminée avec succès.
	ExitErrorGeneric  = 1   // Une erreur non spécifiée est survenue.
	ExitErrorTimeout  = 2   // Le délai d'exécution maximal a été atteint.
	ExitErrorMismatch = 3   // Une incohérence a été détectée entre les résultats de plusieurs algorithmes.
	ExitErrorConfig   = 4   // Une erreur a été détectée dans la configuration fournie par l'utilisateur.
	ExitErrorCanceled = 130 // L'opération a été interrompue par un signal externe (e.g., Ctrl+C).
)

// @const(ProgressBufferMultiplier)
// @description(Facteur multiplicatif pour la mise en tampon (buffering) du canal de communication de la progression.)
// @rationale(L'introduction d'un tampon de communication entre les producteurs (calculateurs) et le consommateur (UI) permet de les découpler. Cela améliore les performances globales en absorbant les variations de latence et en évitant qu'un consommateur lent ne bloque un producteur rapide.)
const ProgressBufferMultiplier = 10

// @struct(AppConfig)
// @description(Agrège l'ensemble des paramètres de configuration de l'application, parsés depuis la ligne de commande.)
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
// @description(Valide la cohérence sémantique de la configuration après le parsing syntaxique.)
// @rationale(Cette méthode implémente une stratégie de "fail-fast", garantissant que les combinaisons de paramètres invalides ou illogiques sont rejetées avant le début de toute exécution coûteuse en ressources.)
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

// @registry(calculatorRegistry)
// @description(Registre centralisant les implémentations disponibles de l'interface `fibonacci.Calculator`.)
// @pattern(Registry)
// @rationale(Ce patron de conception favorise un couplage faible entre le point d'entrée et les implémentations concrètes. Il respecte le Principe Ouvert/Fermé (de SOLID) : pour ajouter un nouvel algorithme, il suffit de l'enregistrer ici sans modifier le reste du code d'orchestration.)
var calculatorRegistry = map[string]fibonacci.Calculator{
	"fast":   fibonacci.NewCalculator(&fibonacci.OptimizedFastDoubling{}),
	"matrix": fibonacci.NewCalculator(&fibonacci.MatrixExponentiation{}),
}

// @function(init)
// @description(Fonction d'initialisation du module, exécutée avant la fonction `main`.)
// @rationale(Utilisée ici pour effectuer une vérification de l'intégrité du registre au démarrage, prévenant ainsi des erreurs d'exécution dues à une configuration de développement incorrecte (e.g., un pointeur nul).)
func init() {
	for name, calc := range calculatorRegistry {
		if calc == nil {
			panic(fmt.Sprintf("Erreur critique d'initialisation : le calculateur enregistré sous le nom '%s' est nil.", name))
		}
	}
}

// @function(getSortedCalculatorKeys)
// @description(Retourne les clés du registre des calculateurs, triées par ordre alphabétique.)
// @rationale(Garantit un comportement déterministe de l'application, notamment dans l'affichage des listes d'algorithmes et dans l'ordre d'exécution du mode "all". Le déterminisme est une propriété essentielle des systèmes robustes.)
func getSortedCalculatorKeys() []string {
	keys := make([]string, 0, len(calculatorRegistry))
	for k := range calculatorRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// @function(main)
// @description(Point d'entrée principal de l'exécutable.)
// @architecture(Son rôle est de servir de pont entre le système d'exploitation et la logique applicative. Il gère les arguments, les flux d'E/S standards, les signaux et les codes de sortie.)
func main() {
	config, err := parseConfig(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(ExitSuccess) // L'utilisateur a demandé l'aide, ce n'est pas une erreur.
		}
		os.Exit(ExitErrorConfig)
	}

	exitCode := run(context.Background(), config, os.Stdout)
	os.Exit(exitCode)
}

// @function(parseConfig)
// @description(Analyse les arguments de la ligne de commande et produit une structure de configuration validée.)
// @pedagogical(Cette fonction est conçue pour être pure et testable. En créant un `flag.NewFlagSet` local et en injectant ses dépendances (les arguments `args` et le flux d'erreur `errorWriter`), on la découple de l'état global du programme, ce qui permet des tests unitaires exhaustifs.)
func parseConfig(programName string, args []string, errorWriter io.Writer) (AppConfig, error) {
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(errorWriter)

	availableAlgos := getSortedCalculatorKeys()
	algoHelp := fmt.Sprintf("Algorithme à utiliser : 'all' (défaut) ou l'un de [%s].", strings.Join(availableAlgos, ", "))

	config := AppConfig{}
	fs.Uint64Var(&config.N, "n", 1000000000, "Indice 'n' du nombre de Fibonacci à calculer.")
	fs.BoolVar(&config.Verbose, "v", false, "Afficher la valeur complète du résultat (peut être très long).")
	fs.BoolVar(&config.Details, "d", false, "Afficher les détails de performance et les métadonnées du résultat.")
	fs.BoolVar(&config.Details, "details", false, "Alias pour -d.")
	fs.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Délai d'exécution maximal pour le calcul.")
	fs.StringVar(&config.Algo, "algo", "all", algoHelp)
	fs.IntVar(&config.Threshold, "threshold", fibonacci.DefaultParallelThreshold, "Seuil (en nombre de bits) pour activer la parallélisation des multiplications.")
	fs.IntVar(&config.FFTThreshold, "fft-threshold", 20000, "Seuil (en nombre de bits) pour utiliser la multiplication par FFT (0 pour désactiver).")
	fs.BoolVar(&config.Calibrate, "calibrate", false, "Exécuter le mode de calibration pour déterminer le seuil de parallélisme optimal.")

	if err := fs.Parse(args); err != nil {
		return AppConfig{}, err // Erreur de parsing syntaxique.
	}

	config.Algo = strings.ToLower(config.Algo)

	if err := config.Validate(availableAlgos); err != nil {
		fmt.Fprintln(errorWriter, "Erreur de configuration :", err)
		fs.Usage()
		return AppConfig{}, errors.New("configuration invalide")
	}

	return config, nil
}

// @struct(CalculationResult)
// @description(Objet de Transfert de Données (DTO) qui encapsule le résultat et les métadonnées d'une exécution de calcul.)
type CalculationResult struct {
	Name     string
	Result   *big.Int
	Duration time.Duration
	Err      error
}

// @function(runCalibration)
// @description(Exécute une suite de benchmarks pour déterminer empiriquement le seuil de parallélisme optimal sur la machine hôte.)
func runCalibration(ctx context.Context, config AppConfig, out io.Writer) int {
	fmt.Fprintln(out, "--- Mode Calibration : Recherche du Seuil de Parallélisme Optimal ---")
	const calibrationN = 10_000_000
	calculator := calculatorRegistry["fast"]
	if calculator == nil {
		fmt.Fprintln(out, "Erreur critique : L'algorithme 'fast' est requis pour la calibration mais n'a pas été trouvé.")
		return ExitErrorGeneric
	}

	thresholdsToTest := []int{0, 256, 512, 1024, 2048, 4096, 8192, 16384} // 0 représente une exécution purement séquentielle.

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
		// La FFT est désactivée pour isoler l'impact du seuil de parallélisme.
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

// @function(run)
// @description(Contient la logique d'orchestration principale de l'application. Cette fonction est pure et testable.)
// @architecture(Cette fonction illustre la composition de contextes pour une gestion robuste de l'arrêt ("graceful shutdown"). Le contexte initial est enrichi successivement avec un timeout et un gestionnaire de signaux OS. L'annulation de l'un de ces contextes se propage à travers toute l'application.)
func run(ctx context.Context, config AppConfig, out io.Writer) int {
	if config.Calibrate {
		return runCalibration(ctx, config, out)
	}

	// Composition des contextes pour la gestion du cycle de vie.
	ctx, cancelTimeout := context.WithTimeout(ctx, config.Timeout)
	defer cancelTimeout()
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// ... Affichage de la configuration ...
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
// @description(Sélectionne les instances de calculateurs à exécuter en fonction de la configuration.)
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
// @description(Orchestre l'exécution concurrente des calculs et de l'interface utilisateur.)
// @pattern(Fan-Out / Fan-In)
// @architecture(Cette fonction implémente le patron "Fan-Out / Fan-In". Une goroutine est lancée pour chaque calcul (fan-out). `errgroup` et `WaitGroup` sont utilisés pour synchroniser l'achèvement de tous les calculs et de la goroutine d'affichage (fan-in).)
func executeCalculations(ctx context.Context, calculators []fibonacci.Calculator, config AppConfig, out io.Writer) []CalculationResult {
	g, ctx := errgroup.WithContext(ctx)
	results := make([]CalculationResult, len(calculators))
	progressChan := make(chan fibonacci.ProgressUpdate, len(calculators)*ProgressBufferMultiplier)

	// Étape de Fan-Out : Lancement des goroutines de calcul.
	for i, calc := range calculators {
		idx, calculator := i, calc // Capture des variables de boucle pour la closure.
		g.Go(func() error {
			startTime := time.Now()
			res, err := calculator.Calculate(ctx, progressChan, idx, config.N, config.Threshold, config.FFTThreshold)
			results[idx] = CalculationResult{
				Name: calculator.Name(), Result: res, Duration: time.Since(startTime), Err: err,
			}
			// Dans un mode de comparaison, l'échec d'un calcul ne doit pas annuler les autres.
			// On retourne donc `nil` pour ne pas déclencher l'annulation du contexte de l'errgroup.
			return nil
		})
	}

	// Lancement de la goroutine de l'interface utilisateur.
	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go cli.DisplayAggregateProgress(&displayWg, progressChan, len(calculators), out)

	// Étape de Fan-In : Synchronisation et attente de la complétion.
	_ = g.Wait()        // Attend la fin de toutes les goroutines de calcul.
	close(progressChan) // Ferme le canal, signalant à l'UI qu'il n'y aura plus de messages.
	displayWg.Wait()    // Attend que la goroutine de l'UI ait terminé son traitement final.

	return results
}

// @function(analyzeComparisonResults)
// @description(Analyse, valide croiséement et affiche les résultats du mode de comparaison.)
func analyzeComparisonResults(results []CalculationResult, config AppConfig, out io.Writer) int {
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil // Les succès sont classés avant les échecs.
		}
		return results[i].Duration < results[j].Duration // Tri secondaire par durée.
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

	// Validation croisée : vérification que tous les résultats réussis sont identiques.
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
	// Affiche les détails du résultat du meilleur algorithme (le premier après le tri).
	cli.DisplayResult(firstValidResult, config.N, 0, config.Verbose, config.Details, out)
	return ExitSuccess
}

// @function(handleCalculationError)
// @description(Interprète une erreur de calcul et retourne le code de sortie système approprié.)
// @pedagogical(L'utilisation de `errors.Is` est la méthode idiomatique et robuste pour inspecter les chaînes d'erreurs en Go. Elle permet de gérer correctement les erreurs qui ont été "enveloppées" (wrapped) par d'autres couches, contrairement à une simple comparaison par `==`.)
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
