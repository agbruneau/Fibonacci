// main.go
//
// Ce programme calcule le n-ième nombre de Fibonacci en utilisant trois algorithmes distincts :
// 1. La formule de Binet (utilisant big.Float pour la haute précision).
// 2. L'algorithme de Doublage Rapide (Fast Doubling).
// 3. L'algorithme d'exponentiation de matrice (Matrice 2x2).
//
// Il exécute ces algorithmes de manière concurrente, affiche leur progression en temps réel,
// et compare leurs temps d'exécution ainsi que leurs résultats.
// Un sync.Pool est utilisé pour réduire les allocations mémoire des objets big.Int.
//
// Utilisation :
//   go run main.go -n <index> -timeout <durée>
// Exemple :
//   go run main.go -n 100000 -timeout 1m

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
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

// fibFunc est un type pour les fonctions calculant les nombres de Fibonacci.
// Il prend un contexte pour l'annulation, un canal pour la progression, l'index n,
// et un pool d'objets big.Int pour la réutilisation de la mémoire.
type fibFunc func(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error)

// task représente une tâche de calcul de Fibonacci à exécuter.
type task struct {
	name string  // Nom de l'algorithme
	fn   fibFunc // Fonction de l'algorithme
}

// result stocke le résultat d'une tâche de calcul.
type result struct {
	name     string        // Nom de l'algorithme
	value    *big.Int      // Valeur de Fibonacci calculée
	duration time.Duration // Durée du calcul
	err      error         // Erreur potentielle
}

// ------------------------------------------------------------
// Gestion de l'Affichage de la Progression
// ------------------------------------------------------------

const progressRefreshInterval = 100 * time.Millisecond

// progressData encapsule les informations de progression pour une tâche.
type progressData struct {
	name string  // Nom de la tâche
	pct  float64 // Pourcentage de progression
}

// progressPrinter gère l'affichage consolidé de la progression pour toutes les tâches.
// Il rafraîchit l'affichage à intervalles réguliers ou lors de la réception de nouvelles données.
//
// Concept:
// Une goroutine dédiée écoute en permanence un canal partagé (progress).
// Elle collecte les pourcentages de chaque tâche et rafraîchit une seule ligne
// sur le terminal pour afficher l'état global. L'astuce `\r` (retour chariot)
// permet de réécrire sur la même ligne, créant une animation fluide de la progression.
func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	for _, name := range taskNames {
		status[name] = 0.0 // Initialise la progression de chaque tâche à 0
	}

	ticker := time.NewTicker(progressRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case p, ok := <-progress:
			if !ok { // Le canal est fermé, fin de la progression.
				printStatus(status, taskNames)
				fmt.Println() // Saut de ligne final
				return
			}
			status[p.name] = p.pct
			printStatus(status, taskNames)

		case <-ticker.C:
			// Rafraîchit l'affichage pour montrer que le programme est toujours actif.
			printStatus(status, taskNames)

		case <-ctx.Done():
			// Le contexte principal est terminé, on cesse l'affichage.
			return
		}
	}
}

// printStatus affiche l'état de progression actuel pour chaque tâche sur une seule ligne.
func printStatus(status map[string]float64, keys []string) {
	var b strings.Builder
	b.WriteString("\r") // Retour chariot pour effacer la ligne précédente

	for i, k := range keys {
		if i > 0 {
			b.WriteString("   ")
		}
		// Formatte la chaîne pour un affichage aligné
		fmt.Fprintf(&b, "%-15s %6.2f%%", k+":", status[k])
	}
	// Ajoute des espaces pour effacer les restes de la ligne précédente si la nouvelle est plus courte
	b.WriteString("                 ")
	fmt.Print(b.String())
}

// ------------------------------------------------------------
// Pool d'objets *big.Int pour la réutilisation mémoire
// ------------------------------------------------------------
//
// Concept d'Optimisation Mémoire (sync.Pool):
// Les calculs pour de grands nombres de Fibonacci nécessitent de manipuler des entiers
// qui dépassent la capacité des types standards (ex: int64). On utilise `big.Int`.
// Le problème: Créer des millions d'objets `big.Int` dans des boucles met une pression
// énorme sur le ramasse-miettes (Garbage Collector), ce qui peut ralentir le programme.
// La solution: Un `sync.Pool` est une sorte de "bac de recyclage" pour les objets.
// Au lieu de créer un nouvel objet à chaque fois, on en demande un au pool.
// Une fois l'utilisation terminée, on le remet dans le pool.
// Cela réduit drastiquement le nombre d'allocations mémoire et améliore les performances.

// newIntPool crée un nouveau sync.Pool pour les objets *big.Int.
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// ------------------------------------------------------------
// Algorithmes de Calcul de Fibonacci
// ------------------------------------------------------------

// fibBinet calcule F(n) en utilisant la formule de Binet.
//
// Concept:
// C'est une formule mathématique directe qui utilise le nombre d'or (φ).
// F(n) = (φ^n - (-φ)^-n) / √5
// Pour un grand n, cela se simplifie en F(n) ≈ round(φ^n / √5).
//
// Implémentation:
// Utilise des nombres à virgule flottante de haute précision (`big.Float`).
// Le calcul principal est une exponentiation binaire de φ pour trouver φ^n efficacement.
//
// Forces/Faiblesses:
// Conceptuellement simple, mais vulnérable aux erreurs de précision inhérentes aux
// calculs en virgule flottante. Souvent moins performant et précis que les méthodes
// basées sur les entiers pour de très grandes valeurs de n.
//
// Note : Cet algorithme n'utilise pas activement le pool de big.Int car il opère sur big.Float.
func fibBinet(ctx context.Context, progress chan<- progressData, n int, _ *sync.Pool) (*big.Int, error) {
	taskName := "Binet"
	if n < 0 {
		return nil, fmt.Errorf("l'index négatif n'est pas supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- progressData{taskName, 100.0}
		}
		return big.NewInt(int64(n)), nil
	}

	// La précision requise augmente avec n.
	// bits pour φ^n ≈ n * log2(φ)
	// On ajoute une marge de sécurité (+10) pour la précision.
	phiVal := (1 + math.Sqrt(5)) / 2
	prec := uint(float64(n)*math.Log2(phiVal) + 10)

	// Fonctions utilitaires pour créer des big.Float avec la bonne précision
	newFloat := func() *big.Float { return new(big.Float).SetPrec(prec) }

	sqrt5 := newFloat().SetUint64(5)
	sqrt5.Sqrt(sqrt5)

	phi := newFloat().SetUint64(1)
	phi.Add(phi, sqrt5)
	phi.Quo(phi, newFloat().SetUint64(2))

	// Calcule φ^n par exponentiation binaire pour minimiser le nombre de multiplications.
	numBitsInN := bits.Len(uint(n))

	phiToN := newFloat().SetInt64(1)
	base := newFloat().Set(phi)

	exponent := uint(n)
	for i := 0; i < numBitsInN; i++ {
		// Vérification coopérative de l'annulation du contexte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if (exponent>>i)&1 == 1 {
			phiToN.Mul(phiToN, base)
		}
		base.Mul(base, base)

		if progress != nil {
			progress <- progressData{taskName, (float64(i+1) / float64(numBitsInN)) * 100.0}
		}
	}

	phiToN.Quo(phiToN, sqrt5)

	// Arrondi au plus proche entier en ajoutant 0.5 avant de tronquer.
	phiToN.Add(phiToN, newFloat().SetFloat64(0.5))

	z := new(big.Int)
	phiToN.Int(z)

	if progress != nil {
		progress <- progressData{taskName, 100.0}
	}
	return z, nil
}

// fibFastDoubling calcule F(n) en utilisant l'algorithme "Fast Doubling".
//
// Concept:
// Un algorithme très efficace basé sur des identités mathématiques qui permettent
// de passer de F(k) et F(k+1) à F(2k) et F(2k+1) en quelques opérations :
// F(2k)   = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)² + F(k+1)²
//
// Implémentation:
// L'algorithme parcourt les bits de l'index `n` de gauche à droite (du plus significatif
// au moins significatif). À chaque étape, il applique les formules de "doublage".
// Si le bit courant de `n` est 1, il fait un pas supplémentaire pour avancer.
//
// Forces/Faiblesses:
// Extrêmement rapide et efficace (complexité en O(log n)). C'est l'un des meilleurs
// algorithmes pour ce problème. Il utilise intensivement le `sync.Pool` pour optimiser
// les allocations de `big.Int`.
func fibFastDoubling(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error) {
	taskName := "Doublage Rapide"
	if n < 0 {
		return nil, fmt.Errorf("l'index négatif n'est pas supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- progressData{taskName, 100.0}
		}
		return big.NewInt(int64(n)), nil
	}

	a := pool.Get().(*big.Int).SetInt64(0) // F(k)
	b := pool.Get().(*big.Int).SetInt64(1) // F(k+1)
	defer pool.Put(a)
	defer pool.Put(b)

	// Variables temporaires pour les calculs, tirées du pool.
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	totalBits := bits.Len(uint(n))
	for i := totalBits - 1; i >= 0; i-- {
		// Vérification coopérative de l'annulation du contexte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Doublement : calcul de F(2k) et F(2k+1) à partir de F(k) et F(k+1)
		// t1 = 2*F(k+1) - F(k)
		t1.Lsh(b, 1)  // t1 = 2*b
		t1.Sub(t1, a) // t1 = 2*b - a
		// t2 = F(k)²
		t2.Mul(a, a) // t2 = a*a
		// F(2k) = F(k) * (2*F(k+1) - F(k))
		a.Mul(a, t1) // a = a * t1
		// t1 = F(k+1)²
		t1.Mul(b, b) // t1 = b*b
		// F(2k+1) = F(k)² + F(k+1)²
		b.Add(t2, t1) // b = t2 + t1

		// Si le i-ème bit de n est 1, on avance d'un pas supplémentaire.
		if (uint(n)>>i)&1 == 1 {
			// On a F(2k) et F(2k+1), on veut F(2k+1) et F(2k+2)
			// t1 = F(2k) + F(2k+1) = F(2k+2)
			t1.Add(a, b)
			// a devient F(2k+1)
			a.Set(b)
			// b devient F(2k+2)
			b.Set(t1)
		}

		if progress != nil {
			progress <- progressData{taskName, (float64(totalBits-i) / float64(totalBits)) * 100.0}
		}
	}

	if progress != nil {
		progress <- progressData{taskName, 100.0}
	}
	// Retourne une nouvelle instance pour ne pas renvoyer un objet du pool qui serait modifié.
	return new(big.Int).Set(a), nil
}

// mat2 représente une matrice 2x2 de *big.Int.
type mat2 struct{ a, b, c, d *big.Int }

// newMat2 crée une nouvelle mat2 dont les composants sont issus du pool.
func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int), b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int), d: pool.Get().(*big.Int),
	}
}

// release remet les composants de la matrice dans le pool.
func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

// set met à jour les valeurs de la matrice cible avec celles d'une autre.
func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// mul effectue la multiplication de deux matrices m1 * m2 et stocke le résultat dans la matrice réceptrice (m).
// Pour des raisons de sécurité, m ne doit pas être un alias de m1 ou m2.
func (m *mat2) mul(m1, m2 *mat2, pool *sync.Pool) {
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	// Calcul de m.a = (m1.a*m2.a) + (m1.b*m2.c)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	m.a.Add(t1, t2)
	// Calcul de m.b = (m1.a*m2.b) + (m1.b*m2.d)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	m.b.Add(t1, t2)
	// Calcul de m.c = (m1.c*m2.a) + (m1.d*m2.c)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	m.c.Add(t1, t2)
	// Calcul de m.d = (m1.c*m2.b) + (m1.d*m2.d)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	m.d.Add(t1, t2)
}

// fibMatrix calcule F(n) par exponentiation de la matrice [[1,1],[1,0]].
//
// Concept:
// Basé sur la propriété que :
// | 1  1 |^k  =  | F(k+1)  F(k)   |
// | 1  0 |     | F(k)    F(k-1) |
// Il suffit donc de calculer la (n-1)-ième puissance de la matrice de base pour trouver F(n)
// dans le coin supérieur gauche (ou droit) du résultat.
//
// Implémentation:
// Le code calcule cette puissance de matrice en utilisant l'exponentiation par carré,
// une technique qui réduit le nombre de multiplications de matrices de O(n) à O(log n).
// La multiplication de matrices 2x2 est implémentée dans la méthode `mat2.mul`.
//
// Forces/Faiblesses:
// Très élégant et également très performant (complexité logarithmique).
// Peut être légèrement plus lent en pratique que le Doublage Rapide à cause du surcoût
// lié à la gestion des 4 éléments de la matrice.
func fibMatrix(ctx context.Context, progress chan<- progressData, n int, pool *sync.Pool) (*big.Int, error) {
	taskName := "Matrice 2x2"
	if n < 0 {
		return nil, fmt.Errorf("l'index négatif n'est pas supporté : %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- progressData{taskName, 100.0}
		}
		return big.NewInt(int64(n)), nil
	}

	// Matrice de résultat, initialisée à l'identité [[1,0],[0,1]]
	res := newMat2(pool)
	defer res.release(pool)
	res.a.SetInt64(1)
	res.b.SetInt64(0)
	res.c.SetInt64(0)
	res.d.SetInt64(1)

	// Matrice de base [[1,1],[1,0]]
	base := newMat2(pool)
	defer base.release(pool)
	base.a.SetInt64(1)
	base.b.SetInt64(1)
	base.c.SetInt64(1)
	base.d.SetInt64(0)

	// Matrice temporaire pour éviter les problèmes d'alias lors de la multiplication
	temp := newMat2(pool)
	defer temp.release(pool)

	exp := uint(n - 1)
	totalSteps := bits.Len(exp)

	for i := 0; exp > 0; i++ {
		// Vérification coopérative de l'annulation du contexte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exp&1 == 1 {
			// res = res * base
			temp.mul(res, base, pool)
			res.set(temp)
		}
		exp >>= 1
		if exp > 0 {
			// base = base * base
			temp.mul(base, base, pool)
			base.set(temp)
		}

		if progress != nil {
			progress <- progressData{taskName, (float64(i+1) / float64(totalSteps)) * 100.0}
		}
	}

	if progress != nil {
		progress <- progressData{taskName, 100.0}
	}

	// Le résultat F(n) est à la position `res.a`
	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Fonction Principale : Le Chef d'Orchestre
// ------------------------------------------------------------
//
// La fonction `main` orchestre tout le processus :
// 1. Elle lit les paramètres de la ligne de commande (`-n` et `-timeout`).
// 2. Elle définit la liste des tâches à exécuter.
// 3. Elle crée un `context` avec un timeout global pour garantir que le programme ne
//    s'exécute pas indéfiniment. Ce contexte est passé à chaque goroutine de calcul
//    pour permettre une annulation coopérative.
// 4. Elle lance la goroutine `progressPrinter` pour l'affichage en temps réel.
// 5. Elle lance une goroutine pour chaque tâche de calcul. L'utilisation de goroutines
//    permet d'exécuter tous les algorithmes en concurrence.
// 6. Elle attend que toutes les tâches se terminent en utilisant un `sync.WaitGroup`.
// 7. Elle ferme les canaux de communication pour signaler aux goroutines réceptrices
//    (comme `progressPrinter`) qu'il n'y aura plus de données.
// 8. Enfin, elle appelle `collectAndDisplayResults` pour analyser et présenter les résultats.
func main() {
	// 1. Lecture des paramètres de la ligne de commande
	nFlag := flag.Int("n", 10000000, "Index n du terme de Fibonacci (entier non-négatif)")
	timeoutFlag := flag.Duration("timeout", 1*time.Minute, "Temps d'exécution maximum global")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag

	if n < 0 {
		log.Fatalf("L'index n doit être supérieur ou égal à 0. Reçu : %d", n)
	}

	// 2. Définition des tâches à exécuter
	definedTasks := []task{
		{"Doublage Rapide", fibFastDoubling},
		{"Matrice 2x2", fibMatrix},
		{"Binet", fibBinet},
	}

	taskNames := make([]string, len(definedTasks))
	for i, t := range definedTasks {
		taskNames[i] = t.name
	}

	log.Printf("Calcul de F(%d) avec un timeout de %v...", n, timeout)
	log.Printf("Algorithmes à exécuter : %s\n", strings.Join(taskNames, ", "))

	// 3. Création du contexte avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Important pour libérer les ressources associées au contexte

	intPool := newIntPool()

	// Canaux pour la communication entre goroutines
	progressAggregatorCh := make(chan progressData, len(definedTasks)*2)
	resultsCh := make(chan result, len(definedTasks))

	// 4. Lancement de l'afficheur de progression
	var wgDisplay sync.WaitGroup
	wgDisplay.Add(1)
	go func() {
		defer wgDisplay.Done()
		progressPrinter(ctx, progressAggregatorCh, taskNames)
	}()

	// 5. Lancement des calculs concurrents
	var wg sync.WaitGroup
	log.Println("Lancement des calculs concurrents...")

	for _, t := range definedTasks {
		wg.Add(1)
		go func(currentTask task) {
			defer wg.Done()
			start := time.Now()
			v, err := currentTask.fn(ctx, progressAggregatorCh, n, intPool)
			duration := time.Since(start)
			resultsCh <- result{currentTask.name, v, duration, err}
		}(t)
	}

	// 6. Attente de la fin de tous les calculs
	wg.Wait()
	log.Println("Calculs terminés.")

	// 7. Fermeture des canaux pour signaler la fin des transmissions
	close(progressAggregatorCh)
	close(resultsCh)

	// Attente de la fin de la goroutine d'affichage
	wgDisplay.Wait()

	// 8. Collecte et affichage des résultats
	collectAndDisplayResults(ctx, resultsCh, n)

	log.Println("Programme terminé.")
}

// collectAndDisplayResults récupère, trie et affiche les résultats des calculs.
//
// Cette fonction est responsable de la présentation finale :
// 1. Elle collecte tous les résultats depuis le canal `resultsCh` jusqu'à sa fermeture.
// 2. Elle trie les résultats: d'abord les succès (par durée croissante), puis les échecs.
// 3. Elle affiche un tableau récapitulatif clair.
// 4. Elle effectue une validation croisée: si plusieurs algorithmes ont réussi,
//    elle vérifie qu'ils ont tous produit le même résultat.
// 5. Elle met en évidence l'algorithme gagnant et affiche des détails sur le nombre calculé.
func collectAndDisplayResults(ctx context.Context, resultsCh <-chan result, n int) {
	results := make([]result, 0)
	// Cette boucle for-range lit depuis le canal jusqu'à ce qu'il soit fermé et vide.
	for r := range resultsCh {
		if r.err != nil {
			// Distingue un timeout d'une autre erreur pour un message plus clair.
			if err := ctx.Err(); err == context.DeadlineExceeded {
				log.Printf("⚠️ Tâche '%s' interrompue par le timeout global après %v", r.name, r.duration.Round(time.Microsecond))
				r.err = err // Assigne l'erreur de contexte pour le tri
			} else {
				log.Printf("❌ Erreur pour la tâche '%s' : %v (durée: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
		results = append(results, r)
	}

	// Tri des résultats : les succès par durée, puis les échecs.
	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true // i est un succès, j est un échec -> i vient avant
		}
		if results[i].err != nil && results[j].err == nil {
			return false // i est un échec, j est un succès -> j vient avant
		}
		// Les deux sont des succès ou les deux sont des échecs -> tri par durée
		return results[i].duration < results[j].duration
	})

	fmt.Println("\n--------------------------- RÉSULTATS ORDONNÉS ---------------------------")
	var firstSuccessfulResult *result
	allValidResultsIdentical := true

	for i, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			if r.err == context.DeadlineExceeded {
				status = "Timeout"
			} else {
				status = fmt.Sprintf("Erreur: %v", r.err)
			}
		} else if r.value != nil {
			// Affiche une version abrégée pour les très grands nombres
			if len(r.value.String()) > 15 {
				valStr = r.value.String()[:5] + "..." + r.value.String()[len(r.value.String())-5:]
			} else {
				valStr = r.value.String()
			}

			// Validation croisée des résultats
			if firstSuccessfulResult == nil {
				firstSuccessfulResult = &results[i]
			} else if r.value.Cmp(firstSuccessfulResult.value) != 0 {
				allValidResultsIdentical = false
			}
		}
		fmt.Printf("%-16s : %-12v [%-14s] Résultat: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	fmt.Println("------------------------------------------------------------------------")

	if firstSuccessfulResult != nil {
		fmt.Printf("\n🏆 Algorithme le plus rapide (ayant réussi) : %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
		printFibResultDetails(firstSuccessfulResult.value, n)
		if allValidResultsIdentical {
			fmt.Println("✅ Tous les résultats valides produits sont identiques.")
		} else {
			fmt.Println("❌ DISCORDANCE ! Les résultats des algorithmes valides diffèrent.")
		}
	} else {
		fmt.Println("\nAucun algorithme n'a pu terminer le calcul avec succès.")
	}
}

// printFibResultDetails affiche des informations détaillées sur le nombre de Fibonacci calculé.
func printFibResultDetails(value *big.Int, n int) {
	if value == nil {
		return
	}

	digits := len(value.Text(10))
	fmt.Printf("Nombre de chiffres de F(%d) : %d\n", n, digits)

	// Utilise la notation scientifique pour les nombres trop grands pour être affichés.
	if digits > 20 {
		floatVal := new(big.Float).SetPrec(uint(digits + 10)).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Valeur (notation scientifique) ≈ %s\n", sci)
	} else {
		fmt.Printf("Valeur = %s\n", value.Text(10))
	}
}