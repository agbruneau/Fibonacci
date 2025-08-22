package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"runtime"
	"sync"
)

// Constantes pour les optimisations.
const (
	// MaxFibUint64 (O1: Fast Path) : L'indice N le plus grand tel que F(N) tienne dans un uint64.
	MaxFibUint64 = 93
	// parallelThreshold (O3: Parallélisation) : Taille minimale en bits pour déclencher le calcul parallèle.
	// Réglé empiriquement pour équilibrer le coût de synchronisation des Goroutines.
	parallelThreshold = 2048
)

func init() {
	// Mise à jour de la description pour refléter les optimisations.
	register("fast-doubling", "Fast Doubling (O(log N) - Optimized: Parallel/FastPath)", &FastDoubling{})
}

// FastDoubling implémente l'algorithme de "Fast Doubling" optimisé.
type FastDoubling struct{}

// workspaceFD contient les variables intermédiaires, permettant leur recyclage.
// Cette structure est adaptée pour supporter la parallélisation (O3) grâce aux variables fk_sq et fkp1_sq distinctes.
type workspaceFD struct {
	a_orig, f2k_term, fk_sq, fkp1_sq, new_a, new_b, t_sum *big.Int
}

func (ws *workspaceFD) acquire(pool *sync.Pool) {
	ws.a_orig = pool.Get().(*big.Int)
	ws.f2k_term = pool.Get().(*big.Int)
	ws.fk_sq = pool.Get().(*big.Int)
	ws.fkp1_sq = pool.Get().(*big.Int)
	ws.new_a = pool.Get().(*big.Int)
	ws.new_b = pool.Get().(*big.Int)
	ws.t_sum = pool.Get().(*big.Int)
}

// release retourne les big.Ints au pool.
func (ws *workspaceFD) release(pool *sync.Pool) {
	// O2: Optimisation de la gestion mémoire.
	// On ne réinitialise pas explicitement les big.Int (SetInt64(0)). C'est un surcoût inutile
	// car l'algorithme écrase toujours la destination avant de la lire.
	pool.Put(ws.a_orig)
	pool.Put(ws.f2k_term)
	pool.Put(ws.fk_sq)
	pool.Put(ws.fkp1_sq)
	pool.Put(ws.new_a)
	pool.Put(ws.new_b)
	pool.Put(ws.t_sum)
}

// calculateSmall (O1: Fast Path) gère N <= 93 en utilisant l'arithmétique native uint64.
func (fd *FastDoubling) calculateSmall(n int) *big.Int {
	if n == 0 {
		return big.NewInt(0)
	}
	// Calcul itératif natif, beaucoup plus rapide que big.Int pour les petits N.
	var a, b uint64 = 0, 1
	for i := 1; i < n; i++ {
		a, b = b, a+b
	}
	return new(big.Int).SetUint64(b)
}

// Calculate exécute l'algorithme Fast Doubling optimisé.
func (fd *FastDoubling) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	// 1. Gestion des cas de base (N < 0, N=0, N=1).
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// 2. O1: Fast Path pour petits N.
	if n <= MaxFibUint64 {
		// S'assurer que la progression finale est envoyée si handleBaseCases ne l'a pas fait.
		reportProgressHelper(progress, 100.0)
		return fd.calculateSmall(n), nil
	}

	// 3. Initialisation pour grands N (Gestion Mémoire).
	a := pool.Get().(*big.Int).SetInt64(0) // F(k)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1) // F(k+1)
	defer pool.Put(b)

	// Récupération du workspace.
	totalBits := bits.Len(uint(n))
	ws := workspaceFD{}
	ws.acquire(pool)
	defer ws.release(pool)

	// 4. Configuration de la boucle et des optimisations.
	// Micro-optimisation: Pré-calcul de l'inverse pour éviter la division dans la boucle.
	invTotalBits := 1.0 / float64(totalBits)

	// O3: Configuration du parallélisme.
	var wg sync.WaitGroup
	// Active le parallélisme uniquement si plus d'un cœur CPU est disponible.
	useParallel := runtime.NumCPU() > 1

	// 5. Boucle principale (Itération MSB vers LSB).
	for i := totalBits - 1; i >= 0; i-- {
		// Gestion de l'annulation (avec encapsulation de l'erreur).
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("calculation canceled: %w", ctx.Err())
		default:
		}

		// --- Étape de Doublage (k -> 2k) ---

		// Sauvegarde F(k) car 'a' sera modifié.
		ws.a_orig.Set(a)

		// 1. F(2k) = F(k) * [2*F(k+1) - F(k)] (Partie Séquentielle)
		ws.f2k_term.Lsh(b, 1)
		ws.f2k_term.Sub(ws.f2k_term, ws.a_orig)
		ws.new_a.Mul(ws.a_orig, ws.f2k_term)

		// 2. F(2k+1) = F(k)^2 + F(k+1)^2 (O3: Partie Parallélisable)

		// Vérifie le seuil de taille (BitLen de b (F(k+1)) est le plus grand) et la disponibilité CPU.
		if useParallel && b.BitLen() > parallelThreshold {
			// Exécution concurrente des deux carrés indépendants.
			wg.Add(2)

			// Goroutine 1: F(k)^2 -> fk_sq
			go func(dest, src *big.Int) {
				defer wg.Done()
				dest.Mul(src, src)
			}(ws.fk_sq, ws.a_orig)

			// Goroutine 2: F(k+1)^2 -> fkp1_sq
			go func(dest, src *big.Int) {
				defer wg.Done()
				dest.Mul(src, src)
			}(ws.fkp1_sq, b)

			wg.Wait()
			ws.new_b.Add(ws.fk_sq, ws.fkp1_sq) // Combinaison des résultats

		} else {
			// Exécution séquentielle pour les petits nombres ou machine monocœur.
			ws.fk_sq.Mul(ws.a_orig, ws.a_orig)
			ws.fkp1_sq.Mul(b, b)
			ws.new_b.Add(ws.fk_sq, ws.fkp1_sq)
		}

		// Mise à jour de l'état (k -> 2k).
		a.Set(ws.new_a)
		b.Set(ws.new_b)

		// --- Étape d'Addition (k -> k+1 si nécessaire) ---
		// Si le i-ème bit de N est 1.
		if (uint(n)>>i)&1 == 1 {
			ws.t_sum.Add(a, b)
			a.Set(b)
			b.Set(ws.t_sum)
		}

		// Rapport de progression (optimisé).
		if progress != nil {
			// Calcul du pourcentage optimisé via multiplication par l'inverse.
			pct := (float64(totalBits-i) * invTotalBits) * 100.0
			reportProgressHelper(progress, pct)
		}
	}

	// CRITICAL: Retourne une COPIE du résultat, car 'a' est retourné au pool via defer.
	return new(big.Int).Set(a), nil
}

// reportProgressHelper est une fonction utilitaire (ajoutée ici car nécessaire pour le snippet)
// pour envoyer la progression de manière non bloquante.
func reportProgressHelper(progressChan chan<- float64, pct float64) {
	if progressChan == nil {
		return
	}
	select {
	case progressChan <- pct:
	default:
		// Drop l'update si le canal est plein.
	}
}
