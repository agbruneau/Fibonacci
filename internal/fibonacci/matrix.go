package fibonacci

import (
	"context"
	"math/big"
	"math/bits"
	"sync"
)

func init() {
	register("matrix", "Matrix 2x2 (O(log N))", &Matrix{})
}

// Matrix implémente l'algorithme d'exponentiation matricielle.
type Matrix struct{}

// mat2 représente une matrice 2x2.
type mat2 struct{ a, b, c, d *big.Int }

// workspaceMat contient les variables intermédiaires pour la multiplication.
type workspaceMat struct {
	t1, t2 *big.Int
}

func (ws *workspaceMat) acquire(pool *sync.Pool) {
	ws.t1 = pool.Get().(*big.Int)
	ws.t2 = pool.Get().(*big.Int)
}

func (ws *workspaceMat) release(pool *sync.Pool) {
	pool.Put(ws.t1)
	pool.Put(ws.t2)
}

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

func (m *mat2) setIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// setFibBase initialise la matrice de base Q = [[1, 1], [1, 0]].
func (m *mat2) setFibBase() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// matMul calcule target = m1 * m2. target ne doit pas être m1 ou m2.
func matMul(target, m1, m2 *mat2, ws *workspaceMat) {
	// a = m1.a*m2.a + m1.b*m2.c
	ws.t1.Mul(m1.a, m2.a)
	ws.t2.Mul(m1.b, m2.c)
	target.a.Add(ws.t1, ws.t2)
	// b = m1.a*m2.b + m1.b*m2.d
	ws.t1.Mul(m1.a, m2.b)
	ws.t2.Mul(m1.b, m2.d)
	target.b.Add(ws.t1, ws.t2)
	// c = m1.c*m2.a + m1.d*m2.c
	ws.t1.Mul(m1.c, m2.a)
	ws.t2.Mul(m1.d, m2.c)
	target.c.Add(ws.t1, ws.t2)
	// d = m1.c*m2.b + m1.d*m2.d
	ws.t1.Mul(m1.c, m2.b)
	ws.t2.Mul(m1.d, m2.d)
	target.d.Add(ws.t1, ws.t2)
}

// Calculate exécute l'algorithme d'exponentiation matricielle.
func (m *Matrix) Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// L'algorithme calcule Q^(N-1).
	exp := uint(n - 1)

	ws := workspaceMat{}
	ws.acquire(pool)
	defer ws.release(pool)

	res := newMat2(pool)
	defer res.release(pool)
	res.setIdentity()
	base := newMat2(pool)
	defer base.release(pool)
	base.setFibBase()
	tempMat := newMat2(pool)
	defer tempMat.release(pool)

	totalSteps := bits.Len(exp)

	// Exponentiation binaire (Square-and-Multiply).
	for stepsDone := 0; exp > 0; stepsDone++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Multiply
		if exp&1 == 1 {
			matMul(tempMat, res, base, &ws)
			res.set(tempMat)
		}

		// Square
		if exp > 1 {
			matMul(tempMat, base, base, &ws)
			base.set(tempMat)
		}

		exp >>= 1

		// Rapport de progression.
		if progress != nil && totalSteps > 0 {
			pct := (float64(stepsDone+1) / float64(totalSteps)) * 100.0
			select {
			case progress <- pct:
			default:
			}
		}
	}

	// F(N) est l'élément (0,0) de la matrice résultante.
	return new(big.Int).Set(res.a), nil
}
