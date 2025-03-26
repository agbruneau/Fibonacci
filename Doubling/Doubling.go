// =============================================================================
// Programme : Calcul ultra-optimisé de Fibonacci(n) en Go
// Auteur    : André-Guy Bruneau // Adapté par l'IA Gemini 2.5 PRo Experimental 03-2025
// Date      : 2025-03-26 // Date de la modification
// Version   : 1.1 // Intégration des raffinements V1.0 (Context propagation, pool opti)
//
// Description :
// Version 1.1 : Intégration des suggestions de raffinement de la V1.0.
// - Optimisation de multiplyMatrices pour utiliser 2 *big.Int temporaires au lieu de 8.
// - Propagation du contexte (context.Context) dans Calculate et fastDoubling pour
//   permettre une annulation réactive (timeout).
// Version 1.0 : Ajout du suivi de progression (basé sur la version 5.2 précédente).
// - fastDoubling affiche le % de bits traités et le temps écoulé ~ toutes les secondes.
// - Utilise math/bits pour déterminer le nombre total d'itérations.
// - Ajout d'une constante ProgressReportInterval.
// ... (Historique précédent omis pour la brièveté) ...
// =============================================================================

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/bits" // Importé pour bits.Len
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// --- Constantes ---
const (
	// ProgressReportInterval : Fréquence de mise à jour de l'indicateur de progression.
	ProgressReportInterval = 1 * time.Second
)

// Configuration des paramètres
type Config struct {
	N               int           // Calculer Fibonacci(N)
	Timeout         time.Duration // Durée max d'exécution
	Precision       int           // Chiffres significatifs après la virgule pour l'affichage scientifique
	Workers         int           // Nombre de threads CPU à utiliser (GOMAXPROCS)
	EnableCache     bool          // Activer le cache simple (utile pour appels répétés)
	EnableProfiling bool          // Activer le profiling CPU/mémoire via pprof
}

// DefaultConfig retourne la configuration par défaut.
func DefaultConfig() Config {
	return Config{
		N:               100000000, // Exemple de grande valeur pour tester la progression
		Timeout:         5 * time.Minute,
		Precision:       10,
		Workers:         runtime.NumCPU(),
		EnableCache:     true,
		EnableProfiling: false, // Mettre à true pour générer les fichiers pprof
	}
}

// Metrics structure pour les métriques de performance.
type Metrics struct {
	StartTime            time.Time
	EndTime              time.Time
	CalculationStartTime time.Time    // Heure de début spécifique au calcul pur
	CalculationEndTime   time.Time    // Heure de fin spécifique au calcul pur
	MatrixOpsCount       atomic.Int64 // Utilisation de atomic.Int64 directement
	CacheHits            atomic.Int64 // Utilisation de atomic.Int64 directement
	TempAllocsAvoided    atomic.Int64 // Utilisation de atomic.Int64 directement
}

// NewMetrics initialise une nouvelle structure Metrics.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddMatrixOps incrémente le compteur d'opérations matricielles de manière atomique.
func (m *Metrics) AddMatrixOps(n int64) {
	m.MatrixOpsCount.Add(n)
}

// AddCacheHit incrémente le compteur de cache hits de manière atomique.
func (m *Metrics) AddCacheHit() {
	m.CacheHits.Add(1)
}

// AddTempAllocsAvoided incrémente le compteur d'allocations temporaires évitées de manière atomique.
func (m *Metrics) AddTempAllocsAvoided(n int64) {
	m.TempAllocsAvoided.Add(n)
}

// CalculationDuration retourne la durée du calcul pur.
func (m *Metrics) CalculationDuration() time.Duration {
	// Gère le cas où le calcul n'a pas encore commencé ou fini
	if m.CalculationStartTime.IsZero() || m.CalculationEndTime.IsZero() {
		return 0
	}
	return m.CalculationEndTime.Sub(m.CalculationStartTime)
}

// FibMatrix représente la matrice 2x2 [[a, b], [c, d]] pour le calcul de Fibonacci.
// Utilise des *big.Int directement pour éviter une indirection supplémentaire.
type FibMatrix struct {
	a, b, c, d *big.Int
}

// FibCalculator encapsule la logique de calcul, le cache, les pools et les métriques.
type FibCalculator struct {
	cache      map[int]*big.Int
	mu         sync.RWMutex // Mutex pour protéger l'accès concurrent au cache
	matrixPool sync.Pool    // Pool pour réutiliser les structures FibMatrix
	bigIntPool sync.Pool    // Pool pour réutiliser les *big.Int temporaires
	config     Config
	metrics    *Metrics
}

// NewFibCalculator crée et initialise un nouveau calculateur Fibonacci.
func NewFibCalculator(cfg Config) *FibCalculator {
	fc := &FibCalculator{
		cache:   make(map[int]*big.Int),
		config:  cfg,
		metrics: NewMetrics(),
	}

	// Initialisation du pool pour FibMatrix
	fc.matrixPool = sync.Pool{
		New: func() interface{} {
			// Initialise une nouvelle matrice avec des big.Int prêts à l'emploi
			return &FibMatrix{
				a: new(big.Int), b: new(big.Int),
				c: new(big.Int), d: new(big.Int),
			}
		},
	}

	// Initialisation du pool pour les *big.Int temporaires
	fc.bigIntPool = sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}

	// Pré-remplissage du cache avec les cas de base si le cache est activé
	if cfg.EnableCache {
		fc.cache[0] = big.NewInt(0)
		fc.cache[1] = big.NewInt(1)
		fc.cache[2] = big.NewInt(1) // F(2)=1
	}

	return fc
}

// getTempBigInt récupère un *big.Int du pool temporaire.
func (fc *FibCalculator) getTempBigInt() *big.Int {
	bi := fc.bigIntPool.Get().(*big.Int)
	fc.metrics.AddTempAllocsAvoided(1)
	// Pas besoin de réinitialiser ici car Mul/Add écrasent la valeur.
	return bi
}

// putTempBigInt remet un *big.Int dans le pool temporaire.
func (fc *FibCalculator) putTempBigInt(bi *big.Int) {
	// Optionnel: Réinitialiser bi à 0 si on veut être très propre,
	// mais généralement non nécessaire car les prochaines opérations écraseront.
	// bi.SetInt64(0)
	fc.bigIntPool.Put(bi)
}

// getMatrix récupère une *FibMatrix du pool.
func (fc *FibCalculator) getMatrix() *FibMatrix {
	m := fc.matrixPool.Get().(*FibMatrix)
	// Pas besoin de réinitialiser les big.Int ici, ils seront écrasés.
	return m
}

// putMatrix remet une *FibMatrix dans le pool.
func (fc *FibCalculator) putMatrix(m *FibMatrix) {
	// Optionnel : réinitialiser les valeurs si nécessaire, mais Mul/Add les écrasent.
	// m.a.SetInt64(0) ...
	fc.matrixPool.Put(m)
}

// multiplyMatrices multiplie deux matrices 2x2 (m1 * m2 = result).
// Utilise **deux** *big.Int temporaires du pool pour minimiser les allocations.
// ATTENTION : result NE DOIT PAS être le même pointeur que m1 ou m2.
func (fc *FibCalculator) multiplyMatrices(m1, m2, result *FibMatrix) {
	// Récupère 2 *big.Int temporaires du pool
	t1 := fc.getTempBigInt()
	t2 := fc.getTempBigInt()

	// Remet les temporaires dans le pool à la fin de la fonction
	defer fc.putTempBigInt(t1)
	defer fc.putTempBigInt(t2)

	// Calcul de result.a = m1.a*m2.a + m1.b*m2.c
	t1.Mul(m1.a, m2.a)   // t1 = m1.a*m2.a
	t2.Mul(m1.b, m2.c)   // t2 = m1.b*m2.c
	result.a.Add(t1, t2) // result.a = t1 + t2

	// Calcul de result.b = m1.a*m2.b + m1.b*m2.d
	t1.Mul(m1.a, m2.b)   // Réutilise t1
	t2.Mul(m1.b, m2.d)   // Réutilise t2
	result.b.Add(t1, t2) // result.b = t1 + t2

	// Calcul de result.c = m1.c*m2.a + m1.d*m2.c
	t1.Mul(m1.c, m2.a)   // Réutilise t1
	t2.Mul(m1.d, m2.c)   // Réutilise t2
	result.c.Add(t1, t2) // result.c = t1 + t2

	// Calcul de result.d = m1.c*m2.b + m1.d*m2.d
	t1.Mul(m1.c, m2.b)   // Réutilise t1
	t2.Mul(m1.d, m2.d)   // Réutilise t2
	result.d.Add(t1, t2) // result.d = t1 + t2

	// Incrémente le compteur global d'opérations matricielles (une multiplication complète)
	// Note: Ceci est compté dans la boucle appelante fastDoubling.
}

// fastDoubling calcule Fibonacci(n) avec l'algorithme de doublement matriciel optimisé,
// affiche la progression et respecte l'annulation via le contexte.
// `calcStartTime` est l'heure de début du calcul pur, utilisée pour afficher le temps écoulé.
// Retourne le résultat et une erreur (nil si succès, ctx.Err() si annulé).
func (fc *FibCalculator) fastDoubling(ctx context.Context, n int, calcStartTime time.Time) (*big.Int, error) {
	// Cas de base gérés avant l'appel (dans Calculate) ou ici pour robustesse
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 || n == 2 {
		return big.NewInt(1), nil
	}

	// --- Initialisation pour la progression ---
	totalIterations := bits.Len(uint(n)) // bits.Len(x) renvoie le nombre de bits pour représenter x
	iterationsDone := 0
	lastReportTime := calcStartTime // Utilise le début du calcul comme référence
	// --- Fin Initialisation Progression ---

	// Récupération des matrices depuis le pool
	matrix := fc.getMatrix()
	result := fc.getMatrix()
	temp := fc.getMatrix() // Matrice temporaire pour les multiplications
	// Utilise defer pour garantir que les matrices sont remises dans le pool,
	// même en cas de retour anticipé (erreur, annulation de contexte).
	defer fc.putMatrix(matrix)
	defer fc.putMatrix(result)
	defer fc.putMatrix(temp)

	// Initialisation de la matrice de base [[1, 1], [1, 0]]
	matrix.a.SetInt64(1)
	matrix.b.SetInt64(1)
	matrix.c.SetInt64(1)
	matrix.d.SetInt64(0)

	// Initialisation de la matrice résultat (Identité [[1, 0], [0, 1]])
	result.a.SetInt64(1)
	result.b.SetInt64(0)
	result.c.SetInt64(0)
	result.d.SetInt64(1)

	m := n // Copie de n pour itérer sur ses bits
	for m > 0 {
		// --- Vérification du contexte ---
		// Vérifie à chaque itération si le contexte a été annulé (timeout, etc.)
		select {
		case <-ctx.Done():
			log.Printf("\nINFO: Calcul interrompu (%v).", ctx.Err())
			fmt.Println() // Assure un saut de ligne après le message d'interruption
			// Les 'defer' s'occuperont de remettre les matrices dans le pool.
			return nil, ctx.Err() // Retourne l'erreur du contexte
		default:
			// Contexte non annulé, continue l'itération
		}
		// --- Fin Vérification du contexte ---

		// --- Logique Fast Doubling ---
		if m&1 != 0 { // Si le bit courant (LSB) de m est 1
			// result = result * matrix (stocké dans temp)
			fc.multiplyMatrices(result, matrix, temp)
			// Échange les pointeurs : result pointe maintenant vers le nouveau résultat (temp)
			result, temp = temp, result
			fc.metrics.AddMatrixOps(1) // Compte une multiplication
		}

		// Mise au carré de la matrice : matrix = matrix * matrix (stocké dans temp)
		fc.multiplyMatrices(matrix, matrix, temp)
		// Échange les pointeurs : matrix pointe maintenant vers le nouveau carré (temp)
		matrix, temp = temp, matrix
		fc.metrics.AddMatrixOps(1) // Compte une multiplication (mise au carré)

		m >>= 1 // Passe au bit suivant de n (division entière par 2)
		// --- Fin Logique Fast Doubling ---

		// --- Mise à jour et Affichage Progression ---
		iterationsDone++
		now := time.Now()
		// Affiche si l'intervalle est dépassé OU si c'est la dernière itération (m==0)
		if now.Sub(lastReportTime) >= ProgressReportInterval || m == 0 {
			elapsed := now.Sub(calcStartTime) // Temps écoulé depuis le début du calcul pur
			var progress float64
			if totalIterations > 0 {
				progress = float64(iterationsDone) / float64(totalIterations) * 100.0
			} else {
				progress = 100.0 // Cas où n=0 ou 1 (ne devrait pas arriver ici mais pour être sûr)
			}

			// Utilise \r pour revenir au début de la ligne
			// Ajoute des espaces pour effacer les restes de lignes précédentes
			fmt.Printf("\rProgress: %.2f%% (%d/%d bits), Elapsed: %v      ",
				progress, iterationsDone, totalIterations, elapsed.Round(time.Millisecond))
			lastReportTime = now
		}
		// --- Fin Progression ---
	}

	// Assurer un saut de ligne après la fin de la barre de progression
	fmt.Println()

	// Le résultat F(n) se trouve dans result.b
	// Crée une nouvelle instance de big.Int pour le résultat final pour éviter
	// les problèmes d'alias si l'appelant modifie la matrice retournée par le pool.
	finalResult := new(big.Int).Set(result.b)
	return finalResult, nil // Retourne le résultat et nil comme erreur
}

// Calculate gère le cache et lance le calcul via fastDoubling.
// Accepte un context.Context pour l'annulation.
// Enregistre également les heures de début/fin du calcul pur dans les métriques.
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("l'index n doit être non-négatif, reçu %d", n)
	}

	// 0. Vérification initiale du contexte avant toute opération coûteuse
	select {
	case <-ctx.Done():
		log.Printf("WARN: Contexte annulé avant même le début du calcul pour F(%d): %v", n, ctx.Err())
		return nil, ctx.Err()
	default:
		// Contexte OK, continuer
	}

	// 1. Vérification du cache (lecture)
	if fc.config.EnableCache {
		fc.mu.RLock() // Verrou en lecture seule
		val, ok := fc.cache[n]
		fc.mu.RUnlock() // Libère le verrou en lecture
		if ok {
			fc.metrics.AddCacheHit()
			now := time.Now()
			fc.metrics.CalculationStartTime = now // Marque comme instantané
			fc.metrics.CalculationEndTime = now
			// Retourne une copie pour éviter que l'appelant modifie la valeur cachée
			return new(big.Int).Set(val), nil
		}
	}

	// 2. Lancement du calcul via Fast Doubling si non trouvé dans le cache
	fc.metrics.CalculationStartTime = time.Now()                            // Enregistrer le début du calcul pur
	result, err := fc.fastDoubling(ctx, n, fc.metrics.CalculationStartTime) // Passe le contexte
	// Si fastDoubling a été annulé ou a échoué
	if err != nil {
		// Ne pas enregistrer EndTime si le calcul a été interrompu
		return nil, fmt.Errorf("le calcul fastDoubling a échoué: %w", err)
	}
	// Si le calcul s'est terminé avec succès
	fc.metrics.CalculationEndTime = time.Now() // Enregistrer la fin du calcul pur

	// 3. Mise en cache du résultat (écriture) si le calcul a réussi
	if fc.config.EnableCache {
		fc.mu.Lock() // Verrou en écriture
		// Vérifie à nouveau au cas où un autre calcul aurait terminé entre temps
		if _, exists := fc.cache[n]; !exists {
			// Met en cache une copie pour la sécurité
			fc.cache[n] = new(big.Int).Set(result)
		}
		fc.mu.Unlock() // Libère le verrou en écriture
	}

	// Retourne le résultat calculé (déjà une nouvelle instance créée dans fastDoubling)
	return result, nil
}

// formatScientific formate un *big.Int en notation scientifique avec une précision donnée.
func formatScientific(num *big.Int, precision int) string {
	if num.Sign() == 0 {
		return "0.0e+0"
	}
	floatPrec := uint(num.BitLen()) + uint(precision) + 10 // Marge de sécurité
	f := new(big.Float).SetPrec(floatPrec).SetInt(num)
	return f.Text('e', precision)
}

func main() {
	cfg := DefaultConfig()
	runtime.GOMAXPROCS(cfg.Workers)
	log.Printf("Configuration: N=%d, Timeout=%v, Workers=%d, Cache=%t, Profiling=%t, Précision Affichage=%d",
		cfg.N, cfg.Timeout, cfg.Workers, cfg.EnableCache, cfg.EnableProfiling, cfg.Precision)

	var fCpu, fMem *os.File
	var err error

	// --- Configuration du Profiling (si activé) ---
	if cfg.EnableProfiling {
		fCpu, err = os.Create("cpu.pprof")
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le fichier de profil CPU: %v", err)
		}
		defer fCpu.Close()
		if err := pprof.StartCPUProfile(fCpu); err != nil {
			log.Fatalf("FATAL: Impossible de démarrer le profil CPU: %v", err)
		}
		defer pprof.StopCPUProfile() // S'assure que le profil est arrêté
		log.Println("INFO: Profiling CPU activé. Fichier: cpu.pprof")

		fMem, err = os.Create("mem.pprof")
		if err != nil {
			log.Printf("WARN: Impossible de créer le fichier de profil mémoire: %v. Profiling mémoire désactivé.", err)
			fMem = nil
		} else {
			defer fMem.Close() // S'assure que le fichier est fermé
			log.Println("INFO: Profiling Mémoire activé. Fichier: mem.pprof (sera écrit à la fin)")
		}
	}

	// --- Initialisation du calculateur ---
	fc := NewFibCalculator(cfg)

	// --- Contexte pour le Timeout ---
	// Crée un contexte qui sera annulé après cfg.Timeout ou si cancel() est appelée.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	// Important: Appeler cancel() garantit que les ressources associées au contexte
	// (comme les goroutines créées par WithTimeout) sont libérées, même si le timeout
	// n'est pas atteint ou si on sort par une autre branche.
	defer cancel()

	// --- Lancement du calcul dans une goroutine séparée ---
	resultChan := make(chan *big.Int, 1)
	errChan := make(chan error, 1)

	go func() {
		log.Printf("INFO: Démarrage du calcul de Fibonacci(%d)... (Timeout: %v)", cfg.N, cfg.Timeout)
		// Passe le contexte à la fonction Calculate
		res, err := fc.Calculate(ctx, cfg.N)

		// Vérifie si l'erreur est due à l'annulation du contexte.
		// Si oui, l'erreur a déjà été loggée dans fastDoubling ou Calculate,
		// et main gérera le ctx.Done(). On ne renvoie pas l'erreur ici pour éviter
		// une double gestion ou un log redondant.
		if err != nil {
			// Si l'erreur N'EST PAS context.Canceled ou context.DeadlineExceeded,
			// alors c'est une autre erreur de calcul qu'il faut signaler.
			if !(err == context.Canceled || err == context.DeadlineExceeded || ctx.Err() != nil) {
				errChan <- fmt.Errorf("erreur interne dans fc.Calculate: %w", err)
			}
			// Si c'était une erreur de contexte, on ne fait rien ici, main s'en chargera via select.
			// La goroutine se termine simplement.
			return
		}

		// Si le calcul a réussi (err == nil)
		fc.metrics.EndTime = time.Now() // Enregistre l'heure de fin globale
		resultChan <- res               // Envoie le résultat
	}()

	// --- Attente du résultat, de l'erreur ou du timeout ---
	var result *big.Int
	log.Println("INFO: En attente du résultat ou du timeout...")

	select {
	case <-ctx.Done():
		// Le contexte a été annulé (timeout ou appel explicite à cancel)
		// fc.metrics.EndTime n'est PAS définie ici car le calcul n'a pas fini normalement.
		// L'erreur ctx.Err() donne la raison (DeadlineExceeded ou Canceled).
		log.Printf("FATAL: Opération annulée ou timeout (%v) dépassé. Raison: %v", cfg.Timeout, ctx.Err())
		// Tente une écriture du profil mémoire même en cas de timeout/annulation si activé
		if cfg.EnableProfiling && fMem != nil {
			log.Println("INFO: Tentative d'écriture du profil mémoire après timeout/annulation...")
			runtime.GC()
			if err := pprof.WriteHeapProfile(fMem); err != nil {
				log.Printf("WARN: Impossible d'écrire le profil mémoire dans %s: %v", fMem.Name(), err)
			} else {
				log.Printf("INFO: Profil mémoire sauvegardé dans %s", fMem.Name())
			}
		}
		os.Exit(1) // Termine avec un code d'erreur

	case err := <-errChan:
		// Une erreur interne (autre que contexte) s'est produite pendant le calcul
		log.Fatalf("FATAL: Erreur interne lors du calcul: %v", err)
		// fc.metrics.EndTime n'est pas définie ici non plus.
		// Les profils seront (tentés d'être) arrêtés/écrits par les defers.

	case result = <-resultChan:
		// Le calcul (ou le hit cache) s'est terminé avec succès
		// fc.metrics.EndTime a été définie dans la goroutine
		calculationDuration := fc.metrics.CalculationDuration()
		log.Printf("INFO: Calcul terminé avec succès. Durée calcul pur: %v", calculationDuration.Round(time.Millisecond))
	}

	// --- Affichage des résultats et métriques (uniquement si succès) ---
	if result != nil {
		fmt.Printf("\n=== Résultats Fibonacci(%d) ===\n", cfg.N)
		totalDuration := fc.metrics.EndTime.Sub(fc.metrics.StartTime)
		calculationDuration := fc.metrics.CalculationDuration()

		fmt.Printf("Temps total d'exécution                     : %v\n", totalDuration.Round(time.Millisecond))
		fmt.Printf("Temps de calcul pur (fastDoubling)          : %v\n", calculationDuration.Round(time.Millisecond))
		fmt.Printf("Opérations matricielles (multiplications)   : %d\n", fc.metrics.MatrixOpsCount.Load())
		if cfg.EnableCache {
			fmt.Printf("Cache hits                                  : %d\n", fc.metrics.CacheHits.Load())
		}
		fmt.Printf("Allocations *big.Int évitées (via pool)   : %d\n", fc.metrics.TempAllocsAvoided.Load())

		fmt.Printf("\nRésultat F(%d) :\n", cfg.N)
		fmt.Printf("  Notation scientifique (~%d chiffres) : %s\n", cfg.Precision, formatScientific(result, cfg.Precision))

		const maxDigitsDisplay = 100
		s := result.String()
		numDigits := len(s)
		fmt.Printf("  Nombre total de chiffres décimaux      : %d\n", numDigits)
		if numDigits <= 2*maxDigitsDisplay {
			fmt.Printf("  Valeur exacte                          : %s\n", s)
		} else {
			fmt.Printf("  Premiers %d chiffres                   : %s...\n", maxDigitsDisplay, s[:maxDigitsDisplay])
			fmt.Printf("  Derniers %d chiffres                   : ...%s\n", maxDigitsDisplay, s[numDigits-maxDigitsDisplay:])
		}

	} else if ctx.Err() == nil {
		// Ce cas ne devrait plus arriver grâce à la gestion d'erreur améliorée,
		// mais on le garde comme filet de sécurité. ctx.Err() != nil si timeout/cancel.
		log.Println("WARN: Le résultat final est nil, mais aucune erreur de contexte détectée (état inattendu).")
	}
	// Si ctx.Err() != nil, le log FATAL a déjà eu lieu.

	// --- Écriture du Profil Mémoire (si activé et fichier créé, et si on n'est pas sorti en FATAL avant) ---
	// Note: Si un FATAL s'est produit (erreur ou timeout), ce code peut ne pas être atteint,
	// mais la tentative d'écriture dans le bloc `case <-ctx.Done():` a été ajoutée.
	// Les `defer` pour fMem.Close() et pprof.StopCPUProfile() s'exécuteront dans tous les cas lors de la sortie de `main`.
	if cfg.EnableProfiling && fMem != nil && result != nil { // N'écrit que si succès ET profiling activé
		log.Println("INFO: Écriture du profil mémoire (heap)...")
		runtime.GC() // Force GC avant le snapshot mémoire
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Impossible d'écrire le profil mémoire dans %s: %v", fMem.Name(), err)
		} else {
			log.Printf("INFO: Profil mémoire sauvegardé dans %s", fMem.Name())
		}
		// fMem.Close() sera appelé par defer
	}

	log.Println("INFO: Programme terminé.")
}
