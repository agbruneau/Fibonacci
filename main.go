/*
================================================================================
Programme : Fibonacci Benchmark (Concurrent & Optimisé)
Date      : 2025-08-01
Version   : 1.1 (Optimisé)

--------------------------------------------------------------------------------
Description :
Ce programme calcule le N-ième nombre de la suite de Fibonacci (F(N)) en utilisant
plusieurs algorithmes distincts (Fast Doubling, Matrix Exponentiation, Formule de Binet,
et Itératif). Il exécute ces algorithmes de manière concurrente pour comparer
leur performance et vérifier l'exactitude de leurs résultats.

Le code est fortement optimisé pour la performance et la gestion mémoire,
notamment via l'utilisation intensive de sync.Pool pour réutiliser les objets
big.Int, réduisant ainsi la pression sur le Garbage Collector (GC).

--------------------------------------------------------------------------------
Instructions d'Utilisation :

1. Prérequis : Go 1.18+

2. Compilation et Exécution :
   go run main.go [flags]
   ou
   go build main.go
   ./main [flags]

3. Flags :
   -n <int>        : L'index N du terme de Fibonacci à calculer. (Défaut: 100000)
   -timeout <dur>  : Le temps d'exécution maximum global (ex: 30s, 1m). (Défaut: 2m)
   -runAlgos <str> : Liste séparée par des virgules des algorithmes à exécuter.
                     (Défaut: all). Les noms sont flexibles (ex: "fast,binet").
   -v              : Mode verbeux. Affiche le nombre complet.

4. Exemples :
   Calculer F(1,000,000) avec un timeout de 30 secondes :
   go run main.go -n 1000000 -timeout 30s

   Calculer F(50,000) en utilisant uniquement Fast Doubling et Matrix :
   go run main.go -n 50000 -runAlgos "fast,matrix"
================================================================================
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	// "math" // Importé dans le code original, mais les constantes nécessaires sont maintenant définies localement.
	"math/big"
	"math/bits"
	"sort"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Types et Structures
// ------------------------------------------------------------

// fibFunc définit la signature que tous les algorithmes de Fibonacci doivent implémenter.
// Elle inclut le contexte pour l'annulation, un canal pour la progression, l'index n,
// et le pool de mémoire.
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

// task représente un algorithme spécifique à exécuter.
type task struct {
	name string  // Nom de l'algorithme
	fn   fibFunc // Pointeur vers la fonction de l'algorithme
}

// result stocke le résultat d'une tâche de calcul.
type result struct {
	name     string
	value    *big.Int
	duration time.Duration
	err      error
}

// Global configuration flags
var verboseFlag bool // Contrôle l'affichage complet du résultat

// ------------------------------------------------------------
// Constantes pour la Formule de Binet
// ------------------------------------------------------------

var (
	// phi (Nombre d'Or) et sqrt5 sont précalculés avec une précision de base élevée.
	phi   = big.NewFloat(1.61803398874989484820458683436563811772030917980576)
	sqrt5 = big.NewFloat(2.23606797749978969640917366873127623544061835961152)
	// log2Phi est utilisé pour estimer la précision nécessaire en bits pour F(N).
	// log2(Phi) ≈ 0.69424
	log2Phi = 0.6942419136306173
)

// ------------------------------------------------------------
// Gestion de l'Affichage de la Progression
// ------------------------------------------------------------

// progressData encapsule les informations de progression envoyées par les tâches.
type progressData struct {
	name string
	pct  float64
}

// progressPrinter gère l'affichage consolidé de la progression de toutes les tâches.
// Il s'exécute dans une goroutine dédiée.
func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	// Ticker pour rafraîchir l'affichage périodiquement.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Initialisation du statut pour toutes les tâches attendues.
	for _, k := range taskNames {
		status[k] = 0.0
	}

	if len(taskNames) == 0 {
		return
	}

	needsUpdate := true // Flag pour savoir si un rafraîchissement est nécessaire.

	for {
		select {
		case <-ctx.Done():
			// AMÉLIORATION: Le contexte est annulé (ex: timeout).
			// Affichage de l'état final et terminaison propre de la goroutine.
			printStatus(status, taskNames)
			fmt.Println("\n(Annulé/Timeout)")
			return
		case p, ok := <-progress:
			if !ok {
				// Le canal est fermé (toutes les tâches sont terminées).
				printStatus(status, taskNames)
				fmt.Println() // Saut de ligne final
				return
			}
			// Mise à jour du statut si le pourcentage a changé.
			if _, exists := status[p.name]; exists && status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			// Rafraîchissement périodique si nécessaire.
			if needsUpdate {
				printStatus(status, taskNames)
				needsUpdate = false
			}
		}
	}
}

// printStatus affiche la progression actuelle sur une seule ligne en écrasant la précédente.
func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r") // Retour chariot pour revenir au début de la ligne.
	var parts []string
	for _, k := range keys {
		v, ok := status[k]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%-18s %6.2f%%", k, v))
	}
	output := strings.Join(parts, " | ")
	// Utilisation de Printf avec un padding large pour effacer complètement la ligne précédente.
	fmt.Printf("%-120s", output)
}

// ------------------------------------------------------------
// Gestion du Pool Mémoire (sync.Pool pour big.Int)
// ------------------------------------------------------------

// newIntPool initialise un sync.Pool pour recycler les objets big.Int.
// Ceci est crucial pour réduire les allocations mémoire et la pression sur le GC.
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// fastDoublingTemps structure pour gérer l'ensemble des variables temporaires
// nécessaires à l'algorithme Fast Doubling.
type fastDoublingTemps struct {
	a_orig, t1, t2, t3, new_a, new_b, t_sum *big.Int
}

// acquire récupère tous les big.Int nécessaires depuis le pool.
func (tmp *fastDoublingTemps) acquire(pool *sync.Pool) {
	tmp.a_orig = pool.Get().(*big.Int)
	tmp.t1 = pool.Get().(*big.Int)
	tmp.t2 = pool.Get().(*big.Int)
	tmp.t3 = pool.Get().(*big.Int)
	tmp.new_a = pool.Get().(*big.Int)
	tmp.new_b = pool.Get().(*big.Int)
	tmp.t_sum = pool.Get().(*big.Int)
}

// release retourne tous les big.Int au pool. Doit être appelé (typiquement via defer).
func (tmp *fastDoublingTemps) release(pool *sync.Pool) {
	pool.Put(tmp.a_orig)
	pool.Put(tmp.t1)
	pool.Put(tmp.t2)
	pool.Put(tmp.t3)
	pool.Put(tmp.new_a)
	pool.Put(tmp.new_b)
	pool.Put(tmp.t_sum)
}

// ------------------------------------------------------------
// Fonctions Utilitaires
// ------------------------------------------------------------

// handleBaseCases gère les cas triviaux (n < 0, n=0, n=1).
func handleBaseCases(n int, progress chan<- float64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("index négatif non supporté: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}
	// Retourne nil, nil si ce n'est pas un cas de base.
	return nil, nil
}

// ------------------------------------------------------------
// Algorithme 1: Fast Doubling (O(log N)) - OPTIMISÉ
// ------------------------------------------------------------

// fibFastDoubling utilise les identités de doublage pour calculer F(N) efficacement.
// F(2k) = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)^2 + F(k+1)^2
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Initialisation : a = F(0) = 0, b = F(1) = 1.
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))
	temps := fastDoublingTemps{}

	// OPTIMISATION MAJEURE : Acquisition des variables temporaires UNE SEULE FOIS, en dehors de la boucle.
	// Le code précédent faisait cela à chaque itération, causant une contention massive sur sync.Pool.
	temps.acquire(pool)
	defer temps.release(pool) // Assure la libération à la fin de la fonction.

	// Itération sur les bits de N, du plus significatif au moins significatif.
	for i := totalBits - 1; i >= 0; i-- {
		// Vérification périodique de l'annulation du contexte (toutes les 5 itérations).
		if i%5 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		temps.a_orig.Set(a) // Sauvegarde F(k)

		// Calcul de F(2k)
		temps.t1.Lsh(b, 1)                      // t1 = 2*F(k+1)
		temps.t1.Sub(temps.t1, temps.a_orig)    // t1 = 2*F(k+1) - F(k)
		temps.new_a.Mul(temps.a_orig, temps.t1) // new_a = F(2k)

		// Calcul de F(2k+1)
		temps.t2.Mul(temps.a_orig, temps.a_orig) // t2 = F(k)^2
		temps.t3.Mul(b, b)                       // t3 = F(k+1)^2
		temps.new_b.Add(temps.t2, temps.t3)      // new_b = F(2k+1)

		a.Set(temps.new_a)
		b.Set(temps.new_b)

		// Si le i-ème bit de n est 1, on avance d'un pas (k -> k+1).
		if (uint(n)>>i)&1 == 1 {
			temps.t_sum.Add(a, b) // t_sum = F(2k+2)
			a.Set(b)              // a devient F(2k+1)
			b.Set(temps.t_sum)    // b devient F(2k+2)
		}

		// Rapport de progression.
		if progress != nil {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}

	// Retourne une copie de 'a', car 'a' appartient au pool et sera réutilisé.
	return new(big.Int).Set(a), nil
}

// ------------------------------------------------------------
// Algorithme 2: Matrix Exponentiation (O(log N))
// ------------------------------------------------------------

// mat2 représente une matrice 2x2 de big.Int.
type mat2 struct {
	a, b, c, d *big.Int
}

// Fonctions utilitaires pour la gestion des matrices dans le pool.
func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int), b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int), d: pool.Get().(*big.Int),
	}
}

func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

// Initialise la matrice identité [[1,0],[0,1]].
func (m *mat2) setIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// Initialise la matrice de base Fibonacci [[1,1],[1,0]].
func (m *mat2) setFibBase() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// Copie les valeurs d'une autre matrice.
func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// matMul calcule target = m1 * m2.
// IMPORTANT: 'target' doit être distinct de m1 et m2 pour éviter l'aliasing (recouvrement mémoire) lors de l'exponentiation.
func matMul(target, m1, m2 *mat2, pool *sync.Pool) {
	// Variables temporaires pour les calculs intermédiaires, prises du pool.
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	// Calcul standard de multiplication matricielle 2x2.
	// target.a = (m1.a*m2.a) + (m1.b*m2.c)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	target.a.Add(t1, t2)
	// target.b = (m1.a*m2.b) + (m1.b*m2.d)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	target.b.Add(t1, t2)
	// target.c = (m1.c*m2.a) + (m1.d*m2.c)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	target.c.Add(t1, t2)
	// target.d = (m1.c*m2.b) + (m1.d*m2.d)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	target.d.Add(t1, t2)
}

// fibMatrix calcule F(N) en utilisant l'exponentiation matricielle binaire.
func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// On calcule M^(n-1). F(n) sera dans res.a.
	exp := uint(n - 1)

	// Matrice résultat (initialisée à l'identité).
	res := newMat2(pool)
	defer res.release(pool)
	res.setIdentity()

	// Matrice de base Fibonacci.
	base := newMat2(pool)
	defer base.release(pool)
	base.setFibBase()

	// Matrice temporaire pour stocker les résultats de multiplication sans aliasing.
	temp := newMat2(pool)
	defer temp.release(pool)

	totalSteps := bits.Len(exp)
	stepsDone := 0

	// Algorithme d'exponentiation binaire (Exponentiation by Squaring).
	for exp > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Si le bit courant est 1, multiplier le résultat par la base actuelle.
		if exp&1 == 1 {
			matMul(temp, res, base, pool)
			res.set(temp) // res = res * base
		}

		// Optimisation : éviter le calcul du carré final inutile.
		if exp > 1 {
			// Mettre la base au carré pour la prochaine itération.
			matMul(temp, base, base, pool)
			base.set(temp) // base = base * base
		}

		exp >>= 1 // Passer au bit suivant.
		stepsDone++

		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}

	// Retourne une copie de res.a.
	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Algorithme 3: Formule de Binet (O(log N), basé sur Float)
// ------------------------------------------------------------

// fibBinet utilise la formule F(n) ≈ round(Phi^n / sqrt(5)).
// Note: Cet algorithme utilise big.Float et non le pool de big.Int.
func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Calcul de la précision nécessaire. Le nombre de bits requis est environ n * log2(Phi).
	// AMÉLIORATION DE ROBUSTESSE : Ajout d'une marge de sécurité de +128 bits pour éviter les erreurs d'arrondi.
	prec := uint(float64(n)*log2Phi + 128)

	// Initialisation des floats avec la précision dynamique.
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	// Calcul de Phi^n en utilisant l'exponentiation binaire (sur big.Float).
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)
	exponent := uint(n)

	numBitsInN := bits.Len(uint(n))
	currentStep := 0

	for exponent > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exponent&1 == 1 {
			result.Mul(result, base)
		}

		if exponent > 1 {
			base.Mul(base, base)
		}
		exponent >>= 1

		currentStep++
		if progress != nil && numBitsInN > 0 {
			// On limite la progression à 99% car la division/arrondi final est rapide.
			progress <- (float64(currentStep) / float64(numBitsInN)) * 99.0
		}
	}

	// (Phi^n) / sqrt(5)
	result.Quo(result, sqrt5Prec)

	// Arrondir à l'entier le plus proche : floor(result + 0.5)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	result.Add(result, half)

	// Conversion en big.Int.
	z := new(big.Int)
	result.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// ------------------------------------------------------------
// Algorithme 4: Itératif Optimisé (O(N))
// ------------------------------------------------------------

// fibIterative est une implémentation O(N) standard, ajoutée comme base de comparaison.
func fibIterative(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Initialisation a=0, b=1.
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	// Variable temporaire pour l'échange (swap), prise du pool.
	temp := pool.Get().(*big.Int)
	defer pool.Put(temp)

	// Définition d'un intervalle de rapport pour éviter de saturer le canal de progression.
	reportInterval := 1000
	if n > 100000 {
		reportInterval = n / 100 // Rapporter environ 100 fois au total.
	}

	for i := 2; i <= n; i++ {
		// Vérification d'annulation et rapport de progression périodiques.
		if i%reportInterval == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				if progress != nil {
					progress <- (float64(i) / float64(n)) * 100.0
				}
			}
		}

		// F(k) = F(k-1) + F(k-2)
		temp.Add(a, b)
		a.Set(b)
		b.Set(temp)
	}

	if progress != nil {
		progress <- 100.0
	}

	// Retourne une copie de 'b'.
	return new(big.Int).Set(b), nil
}

// ------------------------------------------------------------
// Logique d'Exécution Principale (Refactorisée)
// ------------------------------------------------------------

func main() {
	// 1. Analyse des Flags (Parsing)
	nFlag := flag.Int("n", 100000, "Index N du terme de Fibonacci. Défaut 100,000.")
	timeoutFlag := flag.Duration("timeout", 2*time.Minute, "Temps d'exécution maximum global (ex: 1m, 30s).")
	runAlgosFlag := flag.String("runAlgos", "all", "Liste séparée par des virgules des algorithmes (ex: 'binet,fast').")
	flag.BoolVar(&verboseFlag, "v", false, "Sortie verbeuse : affiche le nombre de Fibonacci complet.")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag
	runAlgosStr := strings.ToLower(*runAlgosFlag)

	if n < 0 {
		log.Fatalf("L'index N doit être >= 0. Reçu : %d", n)
	}

	// 2. Définition et Sélection des Tâches
	// Liste canonique des tâches disponibles.
	definedTasks := []task{
		{"Fast-Doubling", fibFastDoubling},
		{"Matrix 2x2", fibMatrix},
		{"Binet (Float)", fibBinet},
		{"Iterative (O(N))", fibIterative},
	}

	// Sélectionne les tâches basées sur l'entrée utilisateur.
	selectedTasks, selectedAlgoNames := selectTasks(definedTasks, runAlgosStr)

	if len(selectedTasks) == 0 {
		log.Println("Aucun algorithme valide sélectionné. Sortie.")
		return
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...", n, timeout)
	log.Printf("Algorithmes : %s", strings.Join(selectedAlgoNames, ", "))

	// 3. Configuration du Contexte, Pool et Canaux
	// Création du contexte global avec timeout.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Assure l'annulation du contexte à la fin du main.

	intPool := newIntPool()
	// Canal bufferisé pour agréger la progression.
	progressAggregatorCh := make(chan progressData, len(selectedTasks)*50)
	// Canal bufferisé pour les résultats.
	resultsCh := make(chan result, len(selectedTasks))

	// 4. Démarrage du Moniteur de Progression
	var printerWg sync.WaitGroup
	printerWg.Add(1)
	go func() {
		defer printerWg.Done()
		// Le printer écoute le contexte pour s'arrêter proprement en cas de timeout.
		progressPrinter(ctx, progressAggregatorCh, selectedAlgoNames)
	}()

	// 5. Lancement des Calculs Concurrents (Fan-out)
	var calculationWg sync.WaitGroup // Pour attendre la fin des calculs.
	var relayWg sync.WaitGroup       // Pour attendre la fin des relais de progression.

	log.Println("Lancement des calculs concurrents...")

	for _, t := range selectedTasks {
		calculationWg.Add(1)
		// Lancement de chaque tâche dans sa propre goroutine.
		go runTask(ctx, t, n, intPool, &calculationWg, &relayWg, progressAggregatorCh, resultsCh)
	}

	// 6. Attente de la complétion (Synchronisation Fan-in)
	calculationWg.Wait()
	// Attendre que les relais aient fini de vider les canaux locaux avant de fermer l'agrégateur.
	relayWg.Wait()
	close(progressAggregatorCh)
	// Attendre que le printer ait affiché le statut final (100% ou timeout).
	printerWg.Wait()

	// 7. Traitement des Résultats
	processResults(resultsCh, selectedTasks, n)

	log.Println("Programme terminé.")
}

// selectTasks analyse l'entrée utilisateur et sélectionne les algorithmes correspondants.
// Utilise une normalisation et une correspondance partielle pour une meilleure UX.
func selectTasks(definedTasks []task, runAlgosStr string) ([]task, []string) {
	// Fonction de normalisation pour rendre la correspondance robuste (insensible à la casse, espaces, tirets).
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		// Supprime les descripteurs pour faciliter la correspondance (ex: "binet" match "Binet (Float)")
		s = strings.ReplaceAll(s, "(float)", "")
		s = strings.ReplaceAll(s, "(o(n))", "")
		s = strings.ReplaceAll(s, "2x2", "")
		return s
	}

	// Map des algorithmes disponibles par leur nom normalisé.
	availableAlgos := make(map[string]task)
	for _, t := range definedTasks {
		availableAlgos[normalize(t.name)] = t
	}

	var selectedTasks []task
	var selectedAlgoNames []string

	// Cas par défaut : exécuter tout.
	if runAlgosStr == "all" || runAlgosStr == "" {
		for _, t := range definedTasks {
			selectedTasks = append(selectedTasks, t)
			selectedAlgoNames = append(selectedAlgoNames, t.name)
		}
		return selectedTasks, selectedAlgoNames
	}

	// Analyse de la liste fournie par l'utilisateur.
	userRequestedAlgoNames := strings.Split(runAlgosStr, ",")
	addedAlgos := make(map[string]bool) // Pour éviter les doublons.

	for _, name := range userRequestedAlgoNames {
		normalizedName := normalize(strings.TrimSpace(name))
		found := false

		// Correspondance partielle (ex: "fast" correspond à "fastdoubling").
		for key, task := range availableAlgos {
			if strings.Contains(key, normalizedName) && normalizedName != "" {
				if !addedAlgos[task.name] {
					selectedTasks = append(selectedTasks, task)
					selectedAlgoNames = append(selectedAlgoNames, task.name)
					addedAlgos[task.name] = true
					found = true
				}
				break // Prend la première correspondance trouvée.
			}
		}

		if !found {
			log.Printf("Avertissement: Algorithme '%s' non reconnu ou non trouvé.", name)
		}
	}
	return selectedTasks, selectedAlgoNames
}

// runTask exécute une tâche Fibonacci spécifique et gère son cycle de vie, incluant le relais de progression.
func runTask(ctx context.Context, currentTask task, n int, pool *sync.Pool, calculationWg *sync.WaitGroup, relayWg *sync.WaitGroup, progressAggregatorCh chan<- progressData, resultsCh chan<- result) {
	defer calculationWg.Done()

	// Canal local pour la progression de CETTE tâche.
	localProgCh := make(chan float64, 10)

	// Goroutine de relais : transfère la progression du canal local vers l'agrégateur central.
	// Cela évite que l'algorithme de calcul ne bloque en essayant d'écrire dans l'agrégateur.
	relayWg.Add(1)
	go func() {
		defer relayWg.Done()
		for p := range localProgCh {
			select {
			case progressAggregatorCh <- progressData{currentTask.name, p}:
			case <-ctx.Done():
				// Arrêt du relais si le contexte global est annulé.
				return
			}
		}
	}()

	// Exécution de l'algorithme.
	start := time.Now()
	v, err := currentTask.fn(ctx, localProgCh, n, pool)
	duration := time.Since(start)
	close(localProgCh) // Fermeture du canal local signale la fin au relais.

	// Envoi du résultat final.
	resultsCh <- result{currentTask.name, v, duration, err}
}

// processResults collecte, trie, affiche et vérifie les résultats.
func processResults(resultsCh chan result, selectedTasks []task, n int) {
	results := make([]result, 0, len(selectedTasks))

	// Collecte des résultats depuis le canal (Fan-in).
	for i := 0; i < len(selectedTasks); i++ {
		r := <-resultsCh
		results = append(results, r)
		// Logging immédiat des erreurs rencontrées.
		if r.err != nil {
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				log.Printf("⚠️ Tâche '%s' a dépassé le délai ou a été annulée après %v", r.name, r.duration.Round(time.Microsecond))
			} else {
				log.Printf("❌ Erreur pour la tâche '%s': %v (durée: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
	}
	close(resultsCh)

	// Tri des résultats : les réussites d'abord, puis par durée croissante.
	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		return results[i].duration < results[j].duration
	})

	// Affichage des Résultats Ordonnés (Tableau Récapitulatif)
	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS ---------------------------")
	for _, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			status = "Erreur"
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				status = "Timeout/Annulé"
			}
		} else if r.value != nil {
			valStr = summarizeBigInt(r.value)
		}
		// Formatage aligné pour une meilleure lisibilité.
		fmt.Printf("%-18s : %-15v [%-16s] Résultat: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	// Vérification et Détails du Résultat Final
	fmt.Println("\n--------------------------- VÉRIFICATION ------------------------------")
	verifyAndPrintDetails(results, n)
}

// summarizeBigInt fournit une représentation courte d'un grand nombre (ex: 12345...67890).
func summarizeBigInt(v *big.Int) string {
	s := v.String()
	if len(s) > 15 {
		return s[:5] + "..." + s[len(s)-5:]
	}
	return s
}

// verifyAndPrintDetails compare les résultats de tous les algorithmes et affiche les détails du plus rapide réussi.
func verifyAndPrintDetails(results []result, n int) {
	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	successfulCount := 0

	for i := range results {
		// Considérer uniquement les résultats valides.
		if results[i].err == nil && results[i].value != nil {
			successfulCount++
			if firstSuccessfulResult == nil {
				// C'est le premier (et donc le plus rapide grâce au tri) résultat réussi.
				firstSuccessfulResult = &results[i]
				fmt.Printf("Algorithme réussi le plus rapide : %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
				printFibResultDetails(firstSuccessfulResult.value, n)
			} else {
				// Comparaison avec le premier résultat réussi.
				if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
					allValidResultsIdentical = false
					log.Printf("⚠️ DIVERGENCE DÉTECTÉE ! Le résultat de '%s' diffère de '%s'.",
						results[i].name, firstSuccessfulResult.name)
				}
			}
		}
	}

	// Conclusion de la vérification.
	if successfulCount > 0 {
		if allValidResultsIdentical {
			fmt.Println("✅ Vérification réussie : Tous les algorithmes terminés ont donné des résultats identiques.")
		} else {
			fmt.Println("❌ Échec de la vérification : Les résultats des algorithmes divergent !")
		}
	} else {
		fmt.Println("❌ Aucun algorithme n'a réussi à terminer le calcul.")
	}
}

// printFibResultDetails affiche des informations détaillées sur le nombre calculé.
func printFibResultDetails(value *big.Int, n int) {
	if value == nil {
		return
	}
	s := value.Text(10) // Conversion en base 10.
	digits := len(s)
	fmt.Printf("Nombre de chiffres dans F(%d) : %d\n", n, digits)

	// Affichage conditionnel basé sur le flag verbose et la longueur du nombre.
	if verboseFlag {
		fmt.Printf("Valeur = %s\n", s)
	} else if digits > 50 {
		// Si trop long, affiche la notation scientifique et les premiers/derniers chiffres.
		floatVal := new(big.Float).SetInt(value)
		sci := floatVal.Text('e', 8) // Notation scientifique avec 8 chiffres de précision.
		fmt.Printf("Valeur ≈ %s\n", sci)
		fmt.Printf("Valeur = %s...%s\n", s[:10], s[len(s)-10:])
	} else {
		fmt.Printf("Valeur = %s\n", s)
	}
}
