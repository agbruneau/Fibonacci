// =============================================================================
// Programme : Calcul ultra-optimisé de Fibonacci(n) en Go
// Auteur    : André-Guy Bruneau // Adapté par l'IA Gemini 2.5 PRo Experimental 03-2025
// Date      : 2025-03-26 // Date de la modification
// Version   : 1.2 // Intégration LRU Cache + String Keys
//
// Description :
// Version 1.2 : Intégration des "Minor Potential Considerations" de la V1.1.
// - Remplacement du cache map[int]*big.Int par un cache LRU (github.com/hashicorp/golang-lru/v2)
//   pour limiter l'utilisation mémoire du cache.
// - Utilisation de string (strconv.Itoa(n)) comme clé de cache pour supprimer la limite théorique de int.
// - Ajout du paramètre Config.CacheSize.
// - Suppression du sync.RWMutex car la bibliothèque LRU gère sa propre synchronisation.
// Version 1.1 : Intégration des suggestions de raffinement de la V1.0.
// - Optimisation de multiplyMatrices pour utiliser 2 *big.Int temporaires.
// - Propagation du contexte (context.Context) dans Calculate et fastDoubling.
// Version 1.0 : Ajout du suivi de progression.
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
	"strconv" // Importé pour convertir int en string pour la clé de cache
	"sync"
	"sync/atomic"
	"time"

	// --- Dépendance Externe ---
	// Nécessite : go get github.com/hashicorp/golang-lru/v2
	lru "github.com/hashicorp/golang-lru/v2"
)

// --- Constantes ---
const (
	// ProgressReportInterval : Fréquence de mise à jour de l'indicateur de progression.
	ProgressReportInterval = 1 * time.Second
)

// Config structure pour les paramètres de configuration.
type Config struct {
	N               int           // Calculer Fibonacci(N)
	Timeout         time.Duration // Durée max d'exécution
	Precision       int           // Chiffres significatifs après la virgule pour l'affichage scientifique
	Workers         int           // Nombre de threads CPU à utiliser (GOMAXPROCS)
	EnableCache     bool          // Activer le cache LRU
	CacheSize       int           // Taille maximale du cache LRU (nombre d'éléments)
	EnableProfiling bool          // Activer le profiling CPU/mémoire via pprof
}

// DefaultConfig retourne la configuration par défaut.
func DefaultConfig() Config {
	return Config{
		N:               10000000, // Exemple de grande valeur pour tester la progression et l'efficacité du cache
		Timeout:         5 * time.Minute,
		Precision:       10,
		Workers:         runtime.NumCPU(),
		EnableCache:     true,
		CacheSize:       2048,  // Taille par défaut du cache LRU, bon compromis mémoire/performance
		EnableProfiling: false, // Mettre à true pour générer les fichiers pprof si besoin d'analyse détaillée
	}
}

// Metrics structure pour collecter les métriques de performance.
type Metrics struct {
	StartTime            time.Time
	EndTime              time.Time
	CalculationStartTime time.Time    // Heure de début spécifique au calcul pur (hors cache hit)
	CalculationEndTime   time.Time    // Heure de fin spécifique au calcul pur
	MatrixOpsCount       atomic.Int64 // Compteur atomique pour les multiplications de matrices
	CacheHits            atomic.Int64 // Compteur atomique pour les accès cache réussis
	TempAllocsAvoided    atomic.Int64 // Compteur atomique pour les allocations évitées via sync.Pool
}

// NewMetrics initialise une nouvelle structure Metrics.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
	// Les autres compteurs atomiques sont initialisés à zéro par défaut.
}

// AddMatrixOps incrémente le compteur d'opérations matricielles de manière atomique.
func (m *Metrics) AddMatrixOps(n int64) {
	m.MatrixOpsCount.Add(n)
}

// AddCacheHit incrémente le compteur de cache hits de manière atomique.
func (m *Metrics) AddCacheHit() {
	m.CacheHits.Add(1)
}

// AddTempAllocsAvoided incrémente le compteur d'allocations temporaires *big.Int évitées de manière atomique.
func (m *Metrics) AddTempAllocsAvoided(n int64) {
	m.TempAllocsAvoided.Add(n)
}

// CalculationDuration retourne la durée du calcul pur (fastDoubling).
// Retourne 0 si le calcul n'a pas été effectué (cache hit) ou n'est pas terminé.
func (m *Metrics) CalculationDuration() time.Duration {
	// Gère le cas où le calcul n'a pas encore commencé ou fini (e.g., cache hit)
	if m.CalculationStartTime.IsZero() || m.CalculationEndTime.IsZero() {
		return 0
	}
	return m.CalculationEndTime.Sub(m.CalculationStartTime)
}

// FibMatrix représente la matrice 2x2 [[a, b], [c, d]] utilisée pour le calcul de Fibonacci.
// Utilise des *big.Int directement pour éviter une indirection et faciliter la manipulation.
type FibMatrix struct {
	a, b, c, d *big.Int
}

// FibCalculator encapsule la logique de calcul, le cache LRU, les pools de ressources,
// la configuration et les métriques.
type FibCalculator struct {
	lruCache   *lru.Cache[string, *big.Int] // Cache LRU thread-safe (clé string, valeur *big.Int)
	matrixPool sync.Pool                    // Pool pour réutiliser les structures FibMatrix
	bigIntPool sync.Pool                    // Pool pour réutiliser les *big.Int temporaires dans les calculs
	config     Config
	metrics    *Metrics
}

// NewFibCalculator crée et initialise un nouveau calculateur Fibonacci en fonction de la configuration.
func NewFibCalculator(cfg Config) *FibCalculator {
	fc := &FibCalculator{
		// lruCache est initialisé conditionnellement ci-dessous
		config:  cfg,
		metrics: NewMetrics(),
	}

	// Initialisation du cache LRU s'il est activé dans la configuration
	if cfg.EnableCache {
		var err error
		// Crée un nouveau cache LRU avec la taille spécifiée.
		// La clé est une string (pour éviter les limites de int), la valeur est *big.Int.
		// La bibliothèque golang-lru/v2 gère la synchronisation interne (thread-safe).
		fc.lruCache, err = lru.New[string, *big.Int](cfg.CacheSize)
		if err != nil {
			// Cette erreur ne devrait se produire que si CacheSize <= 0,
			// ce qui est vérifié dans main() ou géré par les valeurs par défaut.
			log.Fatalf("FATAL: Impossible de créer le cache LRU avec taille %d : %v", cfg.CacheSize, err)
		}
		log.Printf("INFO: Cache LRU activé (taille maximale: %d éléments)", cfg.CacheSize)

		// Pré-remplissage du cache avec les cas de base pour F(0), F(1), F(2).
		// Utilise les clés string correspondantes.
		// Ajoute des copies pour éviter toute modification externe accidentelle.
		fc.lruCache.Add("0", big.NewInt(0))
		fc.lruCache.Add("1", big.NewInt(1))
		fc.lruCache.Add("2", big.NewInt(1)) // F(2) est aussi 1
	} else {
		log.Println("INFO: Cache désactivé par la configuration.")
	}

	// Initialisation du pool pour réutiliser les structures FibMatrix.
	// Réduit les allocations mémoire pour les matrices pendant le calcul.
	fc.matrixPool = sync.Pool{
		New: func() interface{} {
			// Crée une nouvelle FibMatrix avec des *big.Int pré-alloués.
			return &FibMatrix{
				a: new(big.Int), b: new(big.Int),
				c: new(big.Int), d: new(big.Int),
			}
		},
	}

	// Initialisation du pool pour réutiliser les *big.Int temporaires.
	// Crucial pour optimiser multiplyMatrices en évitant des allocations répétées.
	fc.bigIntPool = sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}

	return fc
}

// getTempBigInt récupère un *big.Int depuis le pool temporaire.
// Incrémente le compteur d'allocations évitées.
func (fc *FibCalculator) getTempBigInt() *big.Int {
	bi := fc.bigIntPool.Get().(*big.Int)
	// Le *big.Int récupéré peut contenir une valeur précédente, mais elle sera écrasée par l'appelant.
	fc.metrics.AddTempAllocsAvoided(1)
	return bi
}

// putTempBigInt remet un *big.Int dans le pool temporaire après utilisation.
func (fc *FibCalculator) putTempBigInt(bi *big.Int) {
	// Optionnel : on pourrait remettre à zéro (bi.SetInt64(0)) mais ce n'est pas
	// strictement nécessaire car les utilisateurs écrasent la valeur.
	fc.bigIntPool.Put(bi)
}

// getMatrix récupère une *FibMatrix depuis le pool.
func (fc *FibCalculator) getMatrix() *FibMatrix {
	m := fc.matrixPool.Get().(*FibMatrix)
	// Les champs a, b, c, d de la matrice récupérée peuvent contenir des valeurs précédentes,
	// mais ils seront écrasés par l'appelant (dans fastDoubling ou multiplyMatrices).
	return m
}

// putMatrix remet une *FibMatrix dans le pool après utilisation.
func (fc *FibCalculator) putMatrix(m *FibMatrix) {
	// Optionnel : on pourrait remettre les champs a,b,c,d à zéro, mais ce n'est pas nécessaire.
	fc.matrixPool.Put(m)
}

// multiplyMatrices effectue la multiplication de deux matrices 2x2 (result = m1 * m2).
// Utilise deux *big.Int temporaires récupérés du pool pour minimiser les allocations.
// ATTENTION : Le pointeur 'result' NE DOIT PAS être le même que 'm1' ou 'm2'
//
//	pour éviter d'écraser des valeurs avant qu'elles ne soient lues.
//	Cette condition est respectée dans l'appelant (fastDoubling).
func (fc *FibCalculator) multiplyMatrices(m1, m2, result *FibMatrix) {
	// Récupère deux *big.Int temporaires du pool.
	t1 := fc.getTempBigInt()
	t2 := fc.getTempBigInt()
	// Assure que les temporaires sont remis dans le pool à la fin de la fonction.
	defer fc.putTempBigInt(t1)
	defer fc.putTempBigInt(t2)

	// Calcule result.a = (m1.a * m2.a) + (m1.b * m2.c)
	t1.Mul(m1.a, m2.a)   // t1 = m1.a * m2.a
	t2.Mul(m1.b, m2.c)   // t2 = m1.b * m2.c
	result.a.Add(t1, t2) // result.a = t1 + t2

	// Calcule result.b = (m1.a * m2.b) + (m1.b * m2.d)
	t1.Mul(m1.a, m2.b)   // t1 = m1.a * m2.b
	t2.Mul(m1.b, m2.d)   // t2 = m1.b * m2.d
	result.b.Add(t1, t2) // result.b = t1 + t2

	// Calcule result.c = (m1.c * m2.a) + (m1.d * m2.c)
	t1.Mul(m1.c, m2.a)   // t1 = m1.c * m2.a
	t2.Mul(m1.d, m2.c)   // t2 = m1.d * m2.c
	result.c.Add(t1, t2) // result.c = t1 + t2

	// Calcule result.d = (m1.c * m2.b) + (m1.d * m2.d)
	t1.Mul(m1.c, m2.b)   // t1 = m1.c * m2.b
	t2.Mul(m1.d, m2.d)   // t2 = m1.d * m2.d
	result.d.Add(t1, t2) // result.d = t1 + t2

	// Le comptage des opérations matricielles (fc.metrics.AddMatrixOps)
	// est effectué dans la fonction appelante (fastDoubling) car chaque
	// appel à multiplyMatrices ne correspond pas toujours à une seule
	// "étape" logique de l'algorithme fast doubling.
}

// fastDoubling implémente l'algorithme de calcul de Fibonacci basé sur
// l'exponentiation matricielle rapide (méthode du carré binaire / fast doubling).
// Il vérifie périodiquement l'annulation via le contexte et affiche la progression.
// Retourne F(n) sous forme de *big.Int et une erreur (nil si succès, ctx.Err() si annulé).
func (fc *FibCalculator) fastDoubling(ctx context.Context, n int, calcStartTime time.Time) (*big.Int, error) {
	// Les cas de base (n=0, 1, 2) sont déjà gérés par le cache pré-rempli ou
	// la vérification initiale dans Calculate, mais on les garde ici par robustesse
	// au cas où fastDoubling serait appelé directement ou le cache désactivé.
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 || n == 2 {
		return big.NewInt(1), nil
	}

	// --- Initialisation pour le suivi de progression ---
	// Le nombre d'itérations est lié au nombre de bits de n.
	totalIterations := bits.Len(uint(n)) // bits.Len(0) est 0, bits.Len(1) est 1, etc.
	iterationsDone := 0
	lastReportTime := calcStartTime // Démarre le timer de progression au début du calcul.
	// --- Fin Initialisation Progression ---

	// Récupération des matrices nécessaires depuis le pool pour éviter les allocations.
	matrix := fc.getMatrix() // Matrice de base [[1, 1], [1, 0]] et ses puissances
	result := fc.getMatrix() // Matrice résultat accumulée (commence comme identité)
	temp := fc.getMatrix()   // Matrice temporaire pour stocker les résultats intermédiaires de multiplication
	defer fc.putMatrix(matrix)
	defer fc.putMatrix(result)
	defer fc.putMatrix(temp)

	// Initialisation des matrices :
	// matrix = [[1, 1], [1, 0]]
	matrix.a.SetInt64(1)
	matrix.b.SetInt64(1)
	matrix.c.SetInt64(1)
	matrix.d.SetInt64(0)
	// result = [[1, 0], [0, 1]] (Matrice identité)
	result.a.SetInt64(1)
	result.b.SetInt64(0)
	result.c.SetInt64(0)
	result.d.SetInt64(1)

	// Algorithme Fast Doubling : parcourt les bits de n du plus significatif au moins significatif (implicitement via m >>= 1).
	m := n // Copie de n pour itérer dessus
	for m > 0 {
		// --- Vérification de l'annulation par le contexte ---
		// Vérifie à chaque itération si le contexte a été annulé (timeout, etc.).
		select {
		case <-ctx.Done():
			log.Printf("\nINFO: Calcul interrompu prématurément pour F(%d) car le contexte a été annulé (%v).", n, ctx.Err())
			fmt.Println()         // Assure un saut de ligne après la barre de progression partielle
			return nil, ctx.Err() // Retourne l'erreur du contexte
		default:
			// Contexte non annulé, continuer le calcul.
		}
		// --- Fin Vérification du contexte ---

		// --- Logique principale du Fast Doubling ---
		// Si le bit courant de m est 1 (m & 1 != 0): result = result * matrix
		if m&1 != 0 {
			// Utilise 'temp' pour stocker le nouveau résultat pour éviter d'écraser 'result' pendant que 'multiplyMatrices' le lit.
			fc.multiplyMatrices(result, matrix, temp)
			// Échange les rôles : le résultat est maintenant dans 'temp', l'ancien 'result' devient le nouveau 'temp'.
			result, temp = temp, result
			fc.metrics.AddMatrixOps(1) // Compte une multiplication significative
		}

		// Met la matrice au carré : matrix = matrix * matrix
		// Utilise 'temp' pour stocker le nouveau carré pour éviter d'écraser 'matrix' pendant que 'multiplyMatrices' le lit.
		fc.multiplyMatrices(matrix, matrix, temp)
		// Échange les rôles : le carré est maintenant dans 'temp', l'ancien 'matrix' devient le nouveau 'temp'.
		matrix, temp = temp, matrix
		fc.metrics.AddMatrixOps(1) // Compte une mise au carré significative

		m >>= 1 // Passe au bit suivant (division entière par 2)
		// --- Fin Logique Fast Doubling ---

		// --- Mise à jour et Affichage de la Progression ---
		iterationsDone++ // Compte le nombre de bits traités
		now := time.Now()
		// Affiche la progression toutes les ProgressReportInterval ou à la toute fin (m == 0).
		if now.Sub(lastReportTime) >= ProgressReportInterval || m == 0 {
			elapsed := now.Sub(calcStartTime)
			var progress float64
			if totalIterations > 0 {
				// Calcule le pourcentage de progression basé sur les bits traités.
				progress = float64(iterationsDone) / float64(totalIterations) * 100.0
			} else {
				// Cas où n=0 (totalIterations=0), progression est 100% immédiatement.
				progress = 100.0
			}
			// Utilise \r pour réécrire sur la même ligne dans le terminal.
			fmt.Printf("\rProgression: %.2f%% (%d/%d bits traités), Temps écoulé: %v      ",
				progress, iterationsDone, totalIterations, elapsed.Round(time.Millisecond))
			lastReportTime = now
		}
		// --- Fin Progression ---
	}

	fmt.Println() // Assure un saut de ligne final après la fin de la boucle de progression.

	// Après la boucle, la matrice 'result' contient [[F(n+1), F(n)], [F(n), F(n-1)]].
	// Nous avons besoin de F(n), qui est dans result.b.
	// Crée une copie du résultat pour le retour afin que le *big.Int dans la matrice
	// (qui va retourner au pool) ne soit pas accidentellement modifié par l'appelant.
	finalResult := new(big.Int).Set(result.b)
	return finalResult, nil
}

// Calculate est la fonction principale pour obtenir F(n).
// Elle gère la vérification initiale, la consultation/mise à jour du cache LRU (si activé),
// et délègue le calcul à fastDoubling si nécessaire.
// Accepte un context.Context pour permettre l'annulation (e.g., timeout).
func (fc *FibCalculator) Calculate(ctx context.Context, n int) (*big.Int, error) {
	// Validation de l'entrée : Fibonacci n'est classiquement défini que pour n >= 0.
	if n < 0 {
		return nil, fmt.Errorf("l'index n doit être non-négatif, reçu %d", n)
	}

	// 0. Vérification initiale du contexte avant tout travail.
	select {
	case <-ctx.Done():
		log.Printf("WARN: Contexte déjà annulé avant le début du calcul pour F(%d): %v", n, ctx.Err())
		return nil, ctx.Err()
	default:
		// Contexte OK, continuer.
	}

	// Clé pour le cache LRU : conversion de l'entier n en chaîne de caractères.
	cacheKey := strconv.Itoa(n)

	// 1. Vérification du cache LRU (si activé)
	// Pas besoin de mutex externe, la bibliothèque golang-lru/v2 est thread-safe.
	if fc.config.EnableCache && fc.lruCache != nil {
		if val, ok := fc.lruCache.Get(cacheKey); ok {
			// Cache hit!
			fc.metrics.AddCacheHit()
			now := time.Now()
			// En cas de cache hit, le "calcul" est quasi-instantané.
			fc.metrics.CalculationStartTime = now
			fc.metrics.CalculationEndTime = now
			// Retourne une COPIE de la valeur cachée pour éviter que l'appelant
			// ne modifie accidentellement l'objet *big.Int stocké dans le cache.
			return new(big.Int).Set(val), nil
		}
		// Cache miss, le calcul sera effectué ci-dessous.
	}

	// 2. Lancement du calcul via Fast Doubling (si cache miss ou cache désactivé)
	fc.metrics.CalculationStartTime = time.Now() // Marque le début du calcul réel.
	result, err := fc.fastDoubling(ctx, n, fc.metrics.CalculationStartTime)
	// Gère les erreurs potentielles de fastDoubling (principalement ctx.Err()).
	if err != nil {
		// Si fastDoubling a été interrompu (erreur de contexte), on ne met rien en cache.
		// fc.metrics.CalculationEndTime n'est pas défini ici car le calcul n'a pas abouti.
		// L'erreur sera retournée à l'appelant (main).
		return nil, fmt.Errorf("le calcul fastDoubling pour F(%d) a échoué: %w", n, err)
	}
	// Le calcul a réussi, marquer la fin.
	fc.metrics.CalculationEndTime = time.Now()

	// 3. Mise en cache du résultat dans LRU (si activé et calcul réussi)
	// Add est thread-safe.
	if fc.config.EnableCache && fc.lruCache != nil {
		// Met en cache une COPIE du résultat pour la sécurité, même si fastDoubling
		// retourne déjà une nouvelle instance. C'est une bonne pratique.
		fc.lruCache.Add(cacheKey, new(big.Int).Set(result))
	}

	// Retourne le résultat calculé (qui est déjà une copie créée par fastDoubling).
	return result, nil
}

// formatScientific formate un *big.Int en notation scientifique (e.g., "1.2345e+10")
// avec un nombre spécifié de chiffres après la virgule.
func formatScientific(num *big.Int, precision int) string {
	// Gère le cas spécial de zéro.
	if num.Sign() == 0 {
		return fmt.Sprintf("0.0e+0") // Format cohérent
	}
	// Détermine la précision nécessaire pour big.Float.
	// BitLen donne une idée de la magnitude. Ajoute la précision souhaitée et une marge.
	floatPrec := uint(num.BitLen()) + uint(precision) + 10 // Marge de sécurité pour les arrondis internes
	// Crée un big.Float à partir du big.Int avec la précision calculée.
	f := new(big.Float).SetPrec(floatPrec).SetInt(num)
	// Formate le big.Float en notation scientifique ('e') avec la précision demandée.
	return f.Text('e', precision)
}

// main est le point d'entrée du programme.
func main() {
	// --- Configuration ---
	cfg := DefaultConfig() // Obtient la configuration par défaut

	// --- Validation de la Configuration ---
	if cfg.N < 0 {
		log.Fatalf("FATAL: La valeur N (%d) doit être non-négative.", cfg.N)
	}
	if cfg.CacheSize <= 0 && cfg.EnableCache {
		log.Printf("WARN: CacheSize (%d) est invalide alors que le cache est activé. Désactivation du cache.", cfg.CacheSize)
		cfg.EnableCache = false // Force la désactivation si la taille est invalide
	}
	if cfg.Workers <= 0 {
		log.Printf("WARN: Workers (%d) est invalide. Utilisation de runtime.NumCPU() = %d.", cfg.Workers, runtime.NumCPU())
		cfg.Workers = runtime.NumCPU()
	}

	// Applique le nombre de workers (threads OS) que Go peut utiliser.
	runtime.GOMAXPROCS(cfg.Workers)

	log.Printf("Configuration utilisée: N=%d, Timeout=%v, Workers=%d, Cache=%t, CacheSize=%d, Profiling=%t, Précision Affichage=%d",
		cfg.N, cfg.Timeout, cfg.Workers, cfg.EnableCache, cfg.CacheSize, cfg.EnableProfiling, cfg.Precision)

	var fCpu, fMem *os.File // Fichiers pour le profiling
	var err error

	// --- Configuration du Profiling (si activé) ---
	if cfg.EnableProfiling {
		// Profiling CPU
		fCpu, err = os.Create("cpu.pprof")
		if err != nil {
			log.Fatalf("FATAL: Impossible de créer le fichier de profil CPU 'cpu.pprof': %v", err)
		}
		defer fCpu.Close() // Assure la fermeture du fichier CPU
		if err := pprof.StartCPUProfile(fCpu); err != nil {
			log.Fatalf("FATAL: Impossible de démarrer le profilage CPU: %v", err)
		}
		// pprof.StopCPUProfile() sera appelé via defer juste avant la fin de main.
		defer pprof.StopCPUProfile()
		log.Println("INFO: Profilage CPU activé. Le profil sera écrit dans 'cpu.pprof' à la fin.")

		// Profiling Mémoire (Heap)
		fMem, err = os.Create("mem.pprof")
		if err != nil {
			// Non fatal, on peut continuer sans profiling mémoire.
			log.Printf("WARN: Impossible de créer le fichier de profil mémoire 'mem.pprof': %v. Profilage mémoire désactivé.", err)
			fMem = nil // Assure que fMem est nil si la création échoue
		} else {
			// fMem.Close() sera appelé via defer à la fin de main.
			defer fMem.Close()
			log.Println("INFO: Profilage Mémoire activé. Le profil sera écrit dans 'mem.pprof' à la fin.")
		}
	}

	// --- Initialisation du Calculateur Fibonacci ---
	fc := NewFibCalculator(cfg) // Crée le calculateur avec la configuration validée

	// --- Création du Contexte avec Timeout ---
	// Crée un contexte qui sera automatiquement annulé après la durée cfg.Timeout.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	// cancel() doit être appelé pour libérer les ressources associées au contexte,
	// même si le timeout est atteint ou si l'opération se termine avant.
	defer cancel()

	// --- Lancement du Calcul dans une Goroutine ---
	// Utilise des canaux pour communiquer le résultat ou l'erreur depuis la goroutine.
	resultChan := make(chan *big.Int, 1) // Canal bufferisé pour le résultat (*big.Int)
	errChan := make(chan error, 1)       // Canal bufferisé pour les erreurs

	go func() {
		log.Printf("INFO: Démarrage du calcul de Fibonacci(%d)... (Timeout configuré: %v)", cfg.N, cfg.Timeout)
		// Appelle la méthode Calculate du calculateur, en passant le contexte pour permettre l'annulation.
		res, err := fc.Calculate(ctx, cfg.N)

		// Vérification du résultat et de l'erreur
		if err != nil {
			// Si une erreur s'est produite PENDANT le calcul (pas juste une annulation de contexte)
			// et que le contexte lui-même n'est pas encore annulé (évite un double signalement),
			// alors c'est une erreur interne qu'il faut remonter via errChan.
			// Note: Erreurs de contexte (context.Canceled, context.DeadlineExceeded) sont gérées par le select dans main.
			// Note: `errors.Is(err, context.Canceled)` et `errors.Is(err, context.DeadlineExceeded)` seraient plus robustes que `==`
			//       si l'erreur était enveloppée, mais ici `fc.Calculate` retourne directement `ctx.Err()` ou une erreur enveloppée.
			//       On vérifie aussi ctx.Err() pour les cas limites où l'erreur interne survient juste au moment de l'annulation.
			if !(err == context.Canceled || err == context.DeadlineExceeded || ctx.Err() != nil) {
				// Erreur interne non liée au contexte
				errChan <- fmt.Errorf("erreur interne dans fc.Calculate: %w", err)
			} else {
				// Si c'est une erreur de contexte, le select dans main s'en chargera.
				// Ne rien envoyer sur errChan pour éviter un blocage si main a déjà traité le <-ctx.Done().
			}
			return // Termine la goroutine en cas d'erreur ou d'annulation
		}

		// Si le calcul a réussi sans erreur
		fc.metrics.EndTime = time.Now() // Marque la fin de l'ensemble du processus (calcul + mise en cache)
		resultChan <- res               // Envoie le résultat sur le canal
	}()

	// --- Attente du Résultat, de l'Erreur ou du Timeout ---
	var result *big.Int // Variable pour stocker le résultat final
	log.Println("INFO: En attente du résultat du calcul ou de l'expiration du timeout...")

	select {
	case <-ctx.Done():
		// Le contexte a été annulé (timeout dépassé ou cancel() appelé explicitement).
		// fc.metrics.EndTime n'est pas défini ici car l'opération globale n'a pas réussi.
		log.Printf("FATAL: Opération annulée ou timeout (%v) dépassé. Raison du contexte: %v", cfg.Timeout, ctx.Err())

		// Tentative d'écriture du profil mémoire même en cas de timeout/annulation (si activé).
		if cfg.EnableProfiling && fMem != nil {
			log.Println("INFO: Tentative d'écriture du profil mémoire après timeout/annulation...")
			runtime.GC() // Exécute le garbage collector pour obtenir un profil plus précis de la mémoire en usage.
			if err := pprof.WriteHeapProfile(fMem); err != nil {
				log.Printf("WARN: Impossible d'écrire le profil mémoire dans '%s': %v", fMem.Name(), err)
			} else {
				log.Printf("INFO: Profil mémoire potentiellement partiel sauvegardé dans '%s'", fMem.Name())
			}
		}
		os.Exit(1) // Termine le programme avec un code d'erreur

	case err := <-errChan:
		// Une erreur interne s'est produite pendant le calcul (non liée au contexte).
		// fc.metrics.EndTime n'est pas défini ici.
		log.Fatalf("FATAL: Erreur interne irrécupérable lors du calcul: %v", err)

	case result = <-resultChan:
		// Le calcul s'est terminé avec succès et le résultat est reçu.
		// fc.metrics.EndTime a déjà été défini dans la goroutine.
		calculationDuration := fc.metrics.CalculationDuration()
		log.Printf("INFO: Calcul terminé avec succès. Durée du calcul pur (hors cache): %v", calculationDuration.Round(time.Millisecond))
	}

	// --- Affichage des Résultats et Métriques (uniquement si succès) ---
	if result != nil {
		fmt.Printf("\n=== Résultats pour Fibonacci(%d) ===\n", cfg.N)
		totalDuration := fc.metrics.EndTime.Sub(fc.metrics.StartTime)
		calculationDuration := fc.metrics.CalculationDuration() // Durée du calcul fastDoubling seulement

		fmt.Printf("Temps total d'exécution                     : %v\n", totalDuration.Round(time.Millisecond))
		fmt.Printf("Temps de calcul pur (si effectué)           : %v\n", calculationDuration.Round(time.Millisecond))
		fmt.Printf("Opérations matricielles (multiplications)   : %d\n", fc.metrics.MatrixOpsCount.Load())
		if cfg.EnableCache {
			fmt.Printf("Cache hits                                  : %d\n", fc.metrics.CacheHits.Load())
			// Affiche la taille actuelle du cache par rapport à sa capacité maximale.
			if fc.lruCache != nil {
				fmt.Printf("Cache LRU taille actuelle / max             : %d / %d\n", fc.lruCache.Len(), cfg.CacheSize)
			}
		} else {
			fmt.Println("Cache                                       : Désactivé")
		}
		fmt.Printf("Allocations *big.Int évitées (via pool)   : %d\n", fc.metrics.TempAllocsAvoided.Load())

		fmt.Printf("\nRésultat F(%d) :\n", cfg.N)
		// Affiche en notation scientifique pour donner une idée de l'ordre de grandeur.
		fmt.Printf("  Notation scientifique (~%d chiffres)      : %s\n", cfg.Precision, formatScientific(result, cfg.Precision))

		const maxDigitsDisplay = 100 // Limite pour l'affichage complet/partiel
		s := result.String()         // Convertit le big.Int en chaîne décimale
		numDigits := len(s)
		fmt.Printf("  Nombre total de chiffres décimaux         : %d\n", numDigits)

		// Affiche la valeur exacte si elle n'est pas trop longue, sinon affiche début et fin.
		if numDigits <= 2*maxDigitsDisplay+3 { // +3 pour "..."
			fmt.Printf("  Valeur exacte                             : %s\n", s)
		} else {
			fmt.Printf("  Premiers %d chiffres                      : %s...\n", maxDigitsDisplay, s[:maxDigitsDisplay])
			fmt.Printf("  Derniers %d chiffres                      : ...%s\n", maxDigitsDisplay, s[numDigits-maxDigitsDisplay:])
		}

	} else if ctx.Err() == nil {
		// Ce cas ne devrait normalement pas se produire si la logique select est correcte.
		log.Println("WARN: Le résultat final est nil, mais aucune erreur de contexte ou interne n'a été détectée. État inattendu.")
	}

	// --- Écriture Finale du Profil Mémoire (si succès & activé) ---
	if cfg.EnableProfiling && fMem != nil && result != nil {
		log.Println("INFO: Écriture du profil mémoire final (heap)...")
		runtime.GC() // Exécute le GC avant de capturer le profil pour plus de pertinence.
		if err := pprof.WriteHeapProfile(fMem); err != nil {
			log.Printf("WARN: Impossible d'écrire le profil mémoire final dans '%s': %v", fMem.Name(), err)
		} else {
			log.Printf("INFO: Profil mémoire final sauvegardé dans '%s'", fMem.Name())
		}
	}
	// Les fichiers de profil (fCpu, fMem) sont fermés automatiquement par les appels `defer` en fin de `main`.
	// pprof.StopCPUProfile() est également appelé par `defer`.

	log.Println("INFO: Programme terminé.")
}
