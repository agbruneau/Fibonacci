// Programme : Fibonacci Benchmark (fibbench)
// Description : Point d'entrée principal pour l'outil de benchmark concurrent.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/user/fibbench/internal/config"
	"github.com/user/fibbench/internal/metrics"
	"github.com/user/fibbench/internal/runner"
)

func main() {
	// 1. Chargement de la configuration et initialisation du logger (slog).
	cfg, err := config.Load()
	if err != nil {
		// Utilisation de fmt.Fprintf pour les erreurs avant l'initialisation complète du logger.
		fmt.Fprintf(os.Stderr, "Erreur de configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Gestion des signaux d'interruption (Ctrl+C) pour un arrêt gracieux.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 3. Démarrage du serveur de métriques (optionnel).
	metricsServer := metrics.StartServer(cfg.MetricsPort)

	// 4. Initialisation du Runner.
	r, err := runner.NewRunner(cfg.N, cfg.Timeout, cfg.SelectedAlgos)
	if err != nil {
		slog.Error("Échec de l'initialisation du runner", "error", err)
		os.Exit(1)
	}

	// 5. Exécution du benchmark.
	// Le Runner gère le timeout via le contexte.
	results, err := r.Run(ctx)

	// 6. Traitement et affichage des résultats.
	exitCode := 0
	if err != nil {
		slog.Error("Le benchmark s'est terminé avec une erreur", "error", err)
		// Si le runner retourne une erreur mais qu'il n'y a aucun résultat,
		// on considère cela comme un échec global.
		if len(results) == 0 {
			exitCode = 1
		}
	}

	if len(results) > 0 {
		// Affiche les résultats et détermine le code de sortie en fonction de la cohérence.
		exitCode = processResults(results, cfg)
	} else if err == nil {
		slog.Info("Aucun résultat à afficher.")
	}

	// 7. Arrêt gracieux des services auxiliaires.
	if metricsServer != nil {
		slog.Info("Arrêt du serveur de métriques...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Erreur lors de l'arrêt du serveur de métriques", "error", err)
		}
	}

	// 8. Sortie du programme avec le code approprié.
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// processResults affiche le résumé, effectue la vérification et retourne un code de sortie.
func processResults(results []runner.Result, cfg *config.Config) int {
	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS (Performance) ---------------------------")
	printSummary(results)

	fmt.Println("\n--------------------------- VÉRIFICATION ET DÉTAILS ------------------------------------")
	return verifyAndPrintDetails(results, cfg)
}

// printSummary affiche le tableau récapitulatif des performances.
func printSummary(results []runner.Result) {
	for _, r := range results {
		status, valStr := "OK", "N/A"
		if r.Err != nil {
			status = "Erreur"
			if r.Err == context.DeadlineExceeded || r.Err == context.Canceled {
				status = "Timeout/Annulé"
			}
		} else if r.Value != nil {
			valStr = summarizeBigInt(r.Value)
		}

		fmt.Printf("%-25s : %-15v [%-16s] Résultat: %s\n",
			r.Name,
			r.Duration.Round(time.Microsecond),
			status,
			valStr)
	}
}

// summarizeBigInt crée un résumé concis d'un grand nombre.
func summarizeBigInt(v *big.Int) string {
	s := v.String()
	if len(s) > 24 {
		return s[:10] + "..." + s[len(s)-10:]
	}
	return s
}

// verifyAndPrintDetails compare les résultats, affiche les détails et retourne un code de sortie.
func verifyAndPrintDetails(results []runner.Result, cfg *config.Config) int {
	var fastestSuccess *runner.Result
	allConsistent := true
	successfulCount := 0

	for i := range results {
		r := &results[i]
		if r.Err == nil && r.Value != nil {
			successfulCount++
			if fastestSuccess == nil {
				// C'est le résultat le plus rapide grâce au tri effectué par le runner.
				fastestSuccess = r
				fmt.Printf("Algorithme de référence (le plus rapide) : %s (%v)\n",
					fastestSuccess.Name, fastestSuccess.Duration.Round(time.Microsecond))
				printFibResultDetails(fastestSuccess.Value, cfg)
			} else if r.Value.Cmp(fastestSuccess.Value) != 0 {
				// Divergence détectée.
				allConsistent = false
				slog.Error("DIVERGENCE DÉTECTÉE !",
					"algorithm", r.Name,
					"reference", fastestSuccess.Name)
				if strings.Contains(string(r.Key), "binet") || strings.Contains(string(fastestSuccess.Key), "binet") {
					fmt.Println("  (Note: La formule de Binet peut diverger pour de très grands N en raison des limites de précision flottante).")
				}
			}
		}
	}

	// Conclusion de la vérification.
	if successfulCount > 0 {
		if allConsistent {
			fmt.Println("✅ Vérification réussie : Tous les algorithmes terminés ont donné des résultats identiques.")
			return 0 // Succès
		} else {
			fmt.Println("❌ Échec de la vérification : Les résultats divergent !")
			return 2 // Code de sortie pour divergence
		}
	} else {
		fmt.Println("❌ Aucun algorithme n'a réussi à terminer le calcul dans le délai imparti.")
		return 1 // Code de sortie pour échec de calcul
	}
}

// printFibResultDetails affiche les métadonnées du nombre de Fibonacci calculé.
func printFibResultDetails(value *big.Int, cfg *config.Config) {
	if value == nil {
		return
	}
	s := value.Text(10)
	digits := len(s)
	fmt.Printf("Nombre de chiffres dans F(%d) : %d\n", cfg.N, digits)

	if cfg.Verbose {
		fmt.Printf("Valeur complète :\n%s\n", s)
	} else if digits > 50 {
		// Affichage en notation scientifique et résumé.
		floatVal := new(big.Float).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Valeur ≈ %s\n", sci)
		fmt.Printf("Valeur (extrait) = %s...%s\n", s[:15], s[len(s)-15:])
	} else {
		fmt.Printf("Valeur = %s\n", s)
	}
}
