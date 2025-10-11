// EXPLICATION ACADÉMIQUE :
// Ce fichier implémente le calcul de Fibonacci via l'Exponentiation Matricielle (O(log n)).
// Il illustre l'algorithme d'exponentiation binaire (par la mise au carré), l'optimisation
// mathématique (matrices symétriques), le parallélisme de tâches et la gestion mémoire zéro-allocation.
package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// MatrixExponentiation est une implémentation de l'interface `coreCalculator`.
//
// EXPLICATION ACADÉMIQUE : La Théorie de l'Exponentiation Matricielle
//
// La suite de Fibonacci est une transformation linéaire. La matrice Q = [[1, 1], [1, 0]]
// permet de passer d'un état au suivant :
//
//	[ F(k+1) ] = [ 1  1 ] * [  F(k)  ]
//	[  F(k)  ]   [ 1  0 ]   [ F(k-1) ]
//
// Par conséquent, pour calculer F(n), on peut utiliser la puissance (n-1) de Q :
//
//	[ F(n)  F(n-1) ] = Q^(n-1)
//	[F(n-1) F(n-2) ]
//
// F(n) est l'élément en haut à gauche de Q^(n-1).
// L'exponentiation binaire permet de calculer Q^k en O(log k) multiplications matricielles.
//
// Observation Clé : Symétrie
// Q est symétrique (Q = Qᵀ). Le carré d'une matrice symétrique est aussi symétrique.
// Toutes les puissances de Q sont donc symétriques, permettant une optimisation majeure.
type MatrixExponentiation struct{}

// Name retourne le nom descriptif de l'algorithme et de ses optimisations.
func (c *MatrixExponentiation) Name() string {
	return "Matrix Exponentiation (O(log n) | Parallèle Optimisé | Zéro-Alloc)"
}

// CalculateCore implémente la logique principale de l'algorithme.
func (c *MatrixExponentiation) CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	// Cas de base F(0) = 0.
	if n == 0 {
		return big.NewInt(0), nil
	}

	// --- INITIALISATION & GESTION MÉMOIRE ---

	// Étape 1 : Acquisition d'un état complet depuis le pool (Zéro-Allocation).
	// L'état est réinitialisé par acquireMatrixState (via Reset()).
	state := acquireMatrixState()
	// `defer` garantit le retour de l'état au pool (gestion robuste des ressources).
	defer releaseMatrixState(state)

	// Pour F(n), nous calculons Q^(n-1).
	exponent := n - 1
	numBits := bits.Len64(exponent)

	// Pré-calcul pour le rapport de progression.
	var invNumBits float64
	if numBits > 0 {
		invNumBits = 1.0 / float64(numBits)
	}

	// Détection de la capacité de parallélisme.
	useParallel := runtime.NumCPU() > 1

	// --- ALGORITHME D'EXPONENTIATION BINAIRE (PAR LA MISE AU CARRÉ) ---
	// Parcours des bits de l'exposant (LSB vers MSB).
	// `res` (résultat) : Accumulateur (initialisé à Identité I).
	// `p` (puissance) : Puissances successives Q, Q², Q⁴... (initialisé à Q).
	for i := 0; i < numBits; i++ {
		// Vérification coopérative de l'annulation.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Rapport de progression.
		reporter(float64(i) * invNumBits)

		// ÉTAPE 1 : MULTIPLICATION CONDITIONNELLE (Si le i-ème bit est 1)
		// res = res * p
		if (exponent>>uint(i))&1 == 1 {
			multiplyMatrices(state.tempMatrix, state.res, state.p, state, useParallel, threshold, fftThreshold)

			// OPTIMISATION MAJEURE : Échange de pointeurs (Pointer Swap)
			// Au lieu de copier le contenu de `tempMatrix` vers `res` (coûteux avec big.Int),
			// on échange les pointeurs. `res` pointe vers le nouveau résultat,
			// et l'ancien `res` devient le nouveau buffer temporaire.
			state.res, state.tempMatrix = state.tempMatrix, state.res
		}

		// ÉTAPE 2 : MISE AU CARRÉ
		// p = p * p (pour la prochaine itération)
		// OPTIMISATION : On évite la dernière mise au carré inutile.
		if i < numBits-1 {
			// Puisque `p` est garanti d'être symétrique, on utilise la fonction optimisée
			// (4 multiplications au lieu de 8).
			squareSymmetricMatrix(state.tempMatrix, state.p, state, useParallel, threshold, fftThreshold)
			// Échange de pointeurs (Pointer Swap).
			state.p, state.tempMatrix = state.tempMatrix, state.p
		}
	}

	// Le résultat F(n) est l'élément (0,0) de la matrice résultat Q^(n-1).
	// CRUCIAL : On retourne une NOUVELLE copie. `state.res.a` appartient au pool et sera recyclé.
	return new(big.Int).Set(state.res.a), nil
}

// multiplyMatrices effectue une multiplication standard de deux matrices 2x2.
// C = A * B
func multiplyMatrices(dest, m1, m2 *matrix, state *matrixState, useParallel bool, threshold int, fftThreshold int) {
	// La multiplication standard requiert 8 multiplications d'entiers indépendantes.
	// C[0,0] = A[0,0]*B[0,0] + A[0,1]*B[1,0]  (t1 + t2)
	// C[0,1] = A[0,0]*B[0,1] + A[0,1]*B[1,1]  (t3 + t4)
	// C[1,0] = A[1,0]*B[0,0] + A[1,1]*B[1,0]  (t5 + t6)
	// C[1,1] = A[1,0]*B[0,1] + A[1,1]*B[1,1]  (t7 + t8)

	// Définition des tâches via closures.
	useFFT := fftThreshold > 0 && m1.a.BitLen() > fftThreshold
	var mulFunc func(*big.Int, *big.Int) *big.Int
	if useFFT {
		mulFunc = mulFFT
	} else {
		mulFunc = func(x, y *big.Int) *big.Int {
			return new(big.Int).Mul(x, y)
		}
	}

	tasks := []func(){
		func() { state.t1 = mulFunc(m1.a, m2.a) },
		func() { state.t2 = mulFunc(m1.b, m2.c) },
		func() { state.t3 = mulFunc(m1.a, m2.b) },
		func() { state.t4 = mulFunc(m1.b, m2.d) },
		func() { state.t5 = mulFunc(m1.c, m2.a) },
		func() { state.t6 = mulFunc(m1.d, m2.c) },
		func() { state.t7 = mulFunc(m1.c, m2.b) },
		func() { state.t8 = mulFunc(m1.d, m2.d) },
	}

	// EXPLICATION ACADÉMIQUE : Seuil de Parallélisme Heuristique
	// On vérifie si la taille des nombres dépasse le seuil pour justifier le coût de synchronisation.
	// On utilise `m1.a` comme heuristique car c'est généralement l'élément le plus grand.
	shouldRunInParallel := useParallel && m1.a.BitLen() > threshold
	executeTasks(shouldRunInParallel, tasks)

	// L'assemblage final (additions) est rapide et fait séquentiellement.
	dest.a.Add(state.t1, state.t2)
	dest.b.Add(state.t3, state.t4)
	dest.c.Add(state.t5, state.t6)
	dest.d.Add(state.t7, state.t8)
}

// squareSymmetricMatrix calcule le carré d'une matrice symétrique de manière optimisée.
//
// EXPLICATION DE L'OPTIMISATION (Réduction de 50% des multiplications)
// Soit M une matrice symétrique : M = [[a, b], [b, d]].
//
// M² = [[a²+b², ab+bd],
//
//	[ab+bd, b²+d²]]
//
// M² = [[a²+b², b(a+d)],
//
//	[b(a+d), b²+d²]]
//
// Au lieu de 8 multiplications standard, on n'a besoin que de 4 calculs coûteux :
// a², b², d², et b*(a+d). C'est un gain de performance majeur pour les `big.Int`.
func squareSymmetricMatrix(dest, mat *matrix, state *matrixState, useParallel bool, threshold int, fftThreshold int) {
	// Alias pour les temporaires du pool.
	aSquared := state.t1
	bSquared := state.t2
	dSquared := state.t3
	bTimesAPlusD := state.t4
	aPlusD := state.t5 // Temporaire pour a+d

	// Calcul du terme commun (a+d).
	aPlusD.Add(mat.a, mat.d)

	// Définition des 4 tâches indépendantes.
	useFFT := fftThreshold > 0 && mat.a.BitLen() > fftThreshold

	tasks := []func(){
		// Note : Mul(x, x) est optimisé en interne pour le calcul du carré (squaring).
		func() {
			if useFFT {
				aSquared = mulFFT(mat.a, mat.a)
			} else {
				aSquared.Mul(mat.a, mat.a)
			}
		},
		func() {
			if useFFT {
				bSquared = mulFFT(mat.b, mat.b)
			} else {
				bSquared.Mul(mat.b, mat.b)
			}
		},
		func() {
			if useFFT {
				dSquared = mulFFT(mat.d, mat.d)
			} else {
				dSquared.Mul(mat.d, mat.d)
			}
		},
		func() {
			if useFFT {
				bTimesAPlusD = mulFFT(mat.b, aPlusD)
			} else {
				bTimesAPlusD.Mul(mat.b, aPlusD)
			}
		},
	}

	// Exécution en parallèle si les conditions sont remplies.
	shouldRunInParallel := useParallel && mat.a.BitLen() > threshold
	executeTasks(shouldRunInParallel, tasks)

	// Assemblage de la matrice résultat.
	dest.a.Add(aSquared, bSquared)
	dest.b.Set(bTimesAPlusD)
	dest.c.Set(bTimesAPlusD) // La symétrie est préservée.
	dest.d.Add(bSquared, dSquared)
}

// executeTasks est une fonction utilitaire pour l'exécution parallèle de tâches indépendantes.
// Elle abstrait la logique de synchronisation via `sync.WaitGroup` et optimise l'utilisation
// des goroutines en exécutant la dernière tâche dans la goroutine appelante.
func executeTasks(inParallel bool, tasks []func()) {
	if !inParallel || len(tasks) < 2 {
		// Exécution séquentielle si le parallélisme n'est pas activé ou s'il y a moins de 2 tâches.
		for _, task := range tasks {
			task()
		}
		return
	}

	var wg sync.WaitGroup
	numTasks := len(tasks)
	// On attend seulement N-1 tâches, car la dernière est exécutée dans la goroutine courante.
	wg.Add(numTasks - 1)

	// Lancer N-1 tâches en arrière-plan.
	for i := 0; i < numTasks-1; i++ {
		task := tasks[i] // Capture de la variable de boucle.
		go func() {
			defer wg.Done()
			task()
		}()
	}

	// Exécuter la dernière tâche dans la goroutine courante pour économiser un "spawn".
	tasks[numTasks-1]()

	// Attendre la fin des autres tâches.
	wg.Wait()
}
