package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"sync"
)

func init() {
	register("fast-doubling", "Fast Doubling (O(log N))", &FastDoubling{})
}

// FastDoubling implémente l'algorithme de "Fast Doubling".
type FastDoubling struct{}

// workspaceFD contient les variables intermédiaires, permettant leur recyclage.
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

func (ws *workspaceFD) release(pool *sync.Pool) {
	pool.Put(ws.a_orig)
	pool.Put(ws.f2k_term)
	pool.Put(ws.fk_sq)
	pool.Put(ws.fkp1_sq)
	pool.Put(ws.new_a)
	pool.Put(ws.new_b)
	pool.Put(ws.t_sum)
}

// Calculate exécute l'algorithme Fast Doubling.
func (fd *FastDoubling) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	a := pool.Get().(*big.Int).SetInt64(0) // F(k)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1) // F(k+1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))
	ws := workspaceFD{}
	ws.acquire(pool)
	defer ws.release(pool)

	// Itération sur les bits de N.
	for i := totalBits - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calcul de (F(2k), F(2k+1))

		// F(2k) = F(k) * [2*F(k+1) - F(k)]
		ws.a_orig.Set(a)
		ws.f2k_term.Lsh(b, 1)
		ws.f2k_term.Sub(ws.f2k_term, ws.a_orig)
		ws.new_a.Mul(ws.a_orig, ws.f2k_term)

		// F(2k+1) = F(k)^2 + F(k+1)^2
		ws.fk_sq.Mul(ws.a_orig, ws.a_orig)
		ws.fkp1_sq.Mul(b, b)
		ws.new_b.Add(ws.fk_sq, ws.fkp1_sq)

		a.Set(ws.new_a)
		b.Set(ws.new_b)

		// Si le i-ème bit de N est 1, on avance d'une étape.
		if (uint(n)>>i)&1 == 1 {
			ws.t_sum.Add(a, b)
			a.Set(b)
			b.Set(ws.t_sum)
		}

		// Rapport de progression.
		if progress != nil {
			pct := (float64(totalBits-i) / float64(totalBits)) * 100.0
			select {
			case progress <- pct:
			default:
			}
		}
	}

	return new(big.Int).Set(a), nil
}
