// =============================================================================
// Programme : Calcul de Fibonacci(n) en Go via la formule de Binet
// Auteur    : Adapté par l'IA Gemini 2.5 PRo Experimental 03-2025
// Date      : 2025-03-26
// Version   : 1.0 (Binet)
//
// Description :
// Calcule Fibonacci(n) en utilisant la formule de Binet avec math/big.Float.
// ATTENTION : Cette méthode est principalement à but démonstratif. Elle est
//             généralement MOINS efficace (temps CPU et mémoire) et PLUS
//             complexe à gérer (précision) que l'algorithme Fast Doubling
//             (exponentiation matricielle) pour obtenir des résultats entiers
//             exacts, surtout pour de grandes valeurs de n.
//             Le code Fast Doubling précédent est recommandé pour la performance.
// =============================================================================

package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
	// strconv n'est plus nécessaire car pas de cache avec clé string
	// lru n'est plus nécessaire
	// sync et sync/atomic ne sont plus nécessaires (pas de pools, pas de compteurs atomiques spécifiques)
	// math/bits n'est plus nécessaire
)

// --- Constantes ---
// (ProgressReportInterval n'est pas utilisé ici car FibBinet n'a pas de suivi de progression interne)

// Config structure pour les paramètres de configuration (simplifiée).
type Config struct {
	N               int           // Calculer Fibonacci(N)
	Timeout         time.Duration // Durée max d'exécution globale
	Precision       int           // Chiffres significatifs après la virgule pour l'AFFICHAGE scientifique du résultat
	Workers         int           // Nombre de threads CPU à utiliser (GOMAXPROCS) - Moins pertinent pour Binet sur un seul N
	EnableProfiling bool          // Activer le profiling CPU/mémoire via pprof
}

// DefaultConfig retourne la configuration par défaut pour la version Binet.
func DefaultConfig() Config {
	// ATTENTION: Mettre une valeur de N beaucoup plus petite par défaut
	// car Binet est très lent pour de grands N.
	return Config{
		N: 10000000, // Exemple : petite valeur pour un test rapide
		// N: 1000, // Déjà notablement plus lent
		// N: 100000, // Très lent, consomme beaucoup de mémoire
		// N:               1000000000,       // Extrêmement lent, impraticable !
		Timeout:         5 * time.Minute,  // Garder un timeout généreux
		Precision:       10,               // Pour l'affichage scientifique final
		Workers:         runtime.NumCPU(), // Gardé, mais n'accélère pas un seul calcul Binet
		EnableProfiling: false,            // Mettre à true pour analyser la performance de Binet
	}
}

// Metrics structure pour les métriques de performance (simplifiée).
type Metrics struct {
	StartTime            time.Time
	EndTime              time.Time
	CalculationStartTime time.Time // Heure de début spécifique au calcul FibBinet
	CalculationEndTime   time.Time // Heure de fin spécifique au calcul FibBinet
}

// NewMetrics initialise une nouvelle structure Metrics simplifiée.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// CalculationDuration retourne la durée du calcul FibBinet.
func (m *Metrics) CalculationDuration() time.Duration {
	if m.CalculationStartTime.IsZero() || m.CalculationEndTime.IsZero() {
		return 0
	}
	return m.CalculationEndTime.Sub(m.CalculationStartTime)
}

// FibBinet calcule Fibonacci(n) en utilisant la formule de Binet avec big.Float.
// NOTE : Cette méthode est généralement MOINS efficace et PLUS complexe que
//
//	l'exponentiation matricielle (Fast Doubling) pour obtenir des résultats entiers exacts
//	à cause des exigences de précision très élevées.
//
// NOTE : Cette version n'intègre PAS de vérification de context.Done() pendant le calcul.
//
//	Le timeout global arrêtera le programme, mais n'interrompra pas proprement Binet.
func FibBinet(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("l'index n doit être non-négatif, reçu %d", n)
	}
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}

	startTime := time.Now()
	log.Printf("INFO (Binet): Démarrage calcul pour n=%d", n)

	// --- 1. Déterminer la précision requise (en bits) ---
	// Estimation : n * log2(phi) + marge de sécurité
	prec := uint(math.Ceil(float64(n)*0.69424191) + 128)
	log.Printf("INFO (Binet): Utilisation de précision big.Float = %d bits", prec)

	// --- 2. Constantes avec la précision requise ---
	one := big.NewFloat(1).SetPrec(prec)
	two := big.NewFloat(2).SetPrec(prec)
	five := big.NewFloat(5).SetPrec(prec)
	half := big.NewFloat(0.5).SetPrec(prec)

	// --- 3. Calculer √5 ---
	sqrt5 := new(big.Float).SetPrec(prec)
	sqrt5.Sqrt(five)

	// --- 4. Calculer φ (phi) et ψ (psi) ---
	phi := new(big.Float).SetPrec(prec)
	phi.Add(one, sqrt5)
	phi.Quo(phi, two)

	psi := new(big.Float).SetPrec(prec)
	psi.Sub(one, sqrt5)
	psi.Quo(psi, two)

	// --- 5. Calculer φⁿ et ψⁿ (Exponentiation par carré pour big.Float) ---
	pow := func(base *big.Float, exp int) *big.Float {
		res := big.NewFloat(1).SetPrec(prec)
		tempBase := new(big.Float).Copy(base)
		if exp < 0 {
			oneOverBase := new(big.Float).SetPrec(prec).Quo(one, tempBase)
			tempBase.Copy(oneOverBase)
			exp = -exp
		}
		if exp == 0 {
			return res
		}
		for exp > 0 {
			// TODO: Ajouter vérification ctx.Done() ici si on passe le contexte
			if exp%2 == 1 {
				res.Mul(res, tempBase)
			}
			tempBase.Mul(tempBase, tempBase)
			exp /= 2
		}
		return res
	}

	log.Printf("INFO (Binet): Calcul de phi^%d...", n)
	phiN := pow(phi, n)
	log.Printf("INFO (Binet): Calcul de psi^%d...", n)
	psiN := pow(psi, n) // Note: |psi| < 1, donc psi^n devient très petit

	// --- 6. Calculer (φⁿ - ψⁿ) ---
	numerator := new(big.Float).SetPrec(prec)
	numerator.Sub(phiN, psiN)

	// --- 7. Diviser par √5 ---
	resultFloat := new(big.Float).SetPrec(prec)
	resultFloat.Quo(numerator, sqrt5)

	calculationDuration := time.Since(startTime)
	log.Printf("INFO (Binet): Calcul flottant terminé en %v", calculationDuration)

	// --- 8. Arrondir au plus proche entier (Ajouter 0.5 et tronquer) ---
	resultFloat.Add(resultFloat, half)

	// Convertir en big.Int
	resultInt, accuracy := resultFloat.Int(nil)

	log.Printf("INFO (Binet): Conversion Float -> Int Accuracy: %v", accuracy)
	if accuracy != big.Exact { // On s'attend à ce que ce ne soit pas Exact après +0.5
		// C'est normal, mais on pourrait vouloir logguer si c'est inattendu
	}

	finalDuration := time.Since(startTime)
	log.Printf("INFO (Binet): Calcul total (avec arrondi) terminé en %v", finalDuration)

	return resultInt, nil
}

// formatScientific formate un *big.Int en notation scientifique avec une précision donnée.
// (inchangé par rapport à la version précédente)
func formatScientific(num *big.Int, precision int) string {
	if num.Sign() == 0 {
		return fmt.Sprintf("0.0e+0")
	}
	floatPrec := uint(num.BitLen()) + uint(precision) + 10 // Marge de sécurité
	f := new(big.Float).SetPrec(floatPrec).SetInt(num)
	return f.Text('e', precision)
}

// main est le point d'entrée du programme.
func main() {
	// --- Configuration ---
	cfg := DefaultConfig()

	// --- Validation de la Configuration ---
	if cfg.N < 0 {
		log.Fatalf("FATAL: La valeur N (%d) doit être non-négative.", cfg.N)
	}
	if cfg.Workers <= 0 {
		log.Printf("WARN: Workers (%d) est invalide. Utilisation de runtime.NumCPU() = %d.", cfg.Workers, runtime.NumCPU())
		cfg.Workers = runtime.NumCPU()
	}

	// Applique le nombre de workers. Note : N'accélère pas UN SEUL calcul Binet.
	runtime.GOMAXPROCS(cfg.Workers)

	log.Printf("Configuration (Binet): N=%d, Timeout=%v, Workers=%d, Profiling=%t, Précision Affichage=%d",
		cfg.N, cfg.Timeout, cfg.Workers, cfg.EnableProfiling, cfg.Precision)
	log.Println("ATTENTION: La méthode de Binet est utilisée. Elle est lente et gourmande en mémoire pour N > ~10000.")

	var fCpu, fMem *os.File // Fichiers pour le profiling
	var err error

	// --- Configuration du Profiling (si activé) ---
	if cfg.EnableProfiling {
		// Profiling CPU
		fCpu, err = os.Create("cpu_binet.pprof") // Nom de fichier différent
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le fichier de profil CPU 'cpu_binet.pprof': %v", err)
		}
		defer fCpu.Close()
		if err := pprof.StartCPUProfile(fCpu); err != nil {
			log.Fatalf("FATAL: Impossible de démarrer le profilage CPU: %v", err)
		}
		defer pprof.StopCPUProfile()
		log.Println("INFO: Profilage CPU activé. Profil -> 'cpu_binet.pprof'")

		// Profiling Mémoire (Heap)
		fMem, err = os.Create("mem_binet.pprof") // Nom de fichier différent
		if err != nil {
			log.Printf("WARN: Impossible de créer le fichier de profil mémoire 'mem_binet.pprof': %v. Profilage mémoire désactivé.", err)
			fMem = nil
		} else {
			defer fMem.Close()
			log.Println("INFO: Profilage Mémoire activé. Profil -> 'mem_binet.pprof'")
		}
	}

	// --- Initialisation des Métriques ---
	metrics := NewMetrics()

	// --- Création du Contexte avec Timeout ---
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// --- Lancement du Calcul Binet dans une Goroutine ---
	resultChan := make(chan *big.Int, 1)
	errChan := make(chan error, 1)

	go func() {
		log.Printf("INFO: Démarrage du calcul Binet de Fibonacci(%d)... (Timeout global: %v)", cfg.N, cfg.Timeout)

		// Marquer le début du calcul spécifique
		metrics.CalculationStartTime = time.Now()

		// Appel direct à FibBinet
		// NOTE : On ne passe pas le contexte ici, FibBinet n'est pas conçu pour l'annulation interne.
		res, err := FibBinet(cfg.N)

		// Vérification du résultat et de l'erreur
		if err != nil {
			// Probablement une erreur de configuration (n<0) ou potentiellement
			// une erreur liée à la précision si elle était mal calculée (peu probable ici).
			errChan <- fmt.Errorf("erreur dans FibBinet: %w", err)
			return // Termine la goroutine
		}

		// Si le calcul a réussi
		metrics.CalculationEndTime = time.Now() // Marque la fin du calcul spécifique
		metrics.EndTime = time.Now()            // Marque la fin globale (pour cette goroutine)
		resultChan <- res                       // Envoie le résultat sur le canal
	}()

	// --- Attente du Résultat, de l'Erreur ou du Timeout ---
	var result *big.Int
	log.Println("INFO: En attente du résultat Binet ou du timeout...")

	select {
	case <-ctx.Done():
		// Le contexte a été annulé (timeout global dépassé).
		// Le calcul Binet lui-même n'a pas été proprement interrompu, mais le programme va se terminer.
		metrics.EndTime = time.Now() // Marque la fin au moment du timeout
		log.Printf("FATAL: Opération annulée ou timeout (%v) dépassé pendant le calcul Binet. Raison: %v", cfg.Timeout, ctx.Err())

		// Tentative d'écriture du profil mémoire
		if cfg.EnableProfiling && fMem != nil {
			log.Println("INFO: Tentative d'écriture du profil mémoire après timeout/annulation...")
			runtime.GC()
			if err := pprof.WriteHeapProfile(fMem); err != nil {
				log.Printf("WARN: Impossible d'écrire le profil mémoire dans '%s': %v", fMem.Name(), err)
			} else {
				log.Printf("INFO: Profil mémoire potentiellement partiel sauvegardé dans '%s'", fMem.Name())
			}
		}
		os.Exit(1)

	case err := <-errChan:
		// Une erreur s'est produite pendant FibBinet.
		metrics.EndTime = time.Now() // Marque la fin au moment de l'erreur
		log.Fatalf("FATAL: Erreur lors du calcul Binet: %v", err)

	case result = <-resultChan:
		// Le calcul Binet s'est terminé avec succès.
		// metrics.EndTime a déjà été défini dans la goroutine.
		calculationDuration := metrics.CalculationDuration()
		log.Printf("INFO: Calcul Binet terminé avec succès. Durée calcul pur: %v", calculationDuration.Round(time.Millisecond))
	}

	// --- Affichage des Résultats et Métriques (si succès) ---
	if result != nil {
		fmt.Printf("\n=== Résultats Binet pour Fibonacci(%d) ===\n", cfg.N)
		totalDuration := metrics.EndTime.Sub(metrics.StartTime)
		calculationDuration := metrics.CalculationDuration()

		fmt.Printf("Temps total d'exécution                     : %v\n", totalDuration.Round(time.Millisecond))
		fmt.Printf("Temps de calcul pur (FibBinet)              : %v\n", calculationDuration.Round(time.Millisecond))
		// Pas d'autres métriques spécifiques à Binet à afficher ici

		fmt.Printf("\nRésultat F(%d) :\n", cfg.N)
		fmt.Printf("  Notation scientifique (~%d chiffres)      : %s\n", cfg.Precision, formatScientific(result, cfg.Precision))

		const maxDigitsDisplay = 100
		s := result.String()
		numDigits := len(s)
		fmt.Printf("  Nombre total de chiffres décimaux         : %d\n", numDigits)

		if numDigits <= 2*maxDigitsDisplay+3 {
			fmt.Printf("  Valeur exacte                             : %s\n", s)
		} else {
			fmt.Printf("  Premiers %d chiffres                      : %s...\n", maxDigitsDisplay, s[:maxDigitsDisplay])
			fmt.Printf("  Derniers %d chiffres                      : ...%s\n", maxDigitsDisplay, s[numDigits-maxDigitsDisplay:])
		}

	} else if ctx.Err() == nil {
		log.Println("WARN: Le résultat final est nil, mais aucune erreur détectée. État inattendu.")
	}

	// --- Écriture Finale du Profil Mémoire (si succès & activé) ---
	if cfg.EnableProfiling && fMem != nil && result != nil {
		log.Println("INFO: Écriture du profil mémoire final (heap)...")
		runtime.GC()
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Impossible d'écrire le profil mémoire final dans '%s': %v", fMem.Name(), err)
		} else {
			log.Printf("INFO: Profil mémoire final sauvegardé dans '%s'", fMem.Name())
		}
	}

	log.Println("INFO: Programme (Binet) terminé.")
}
