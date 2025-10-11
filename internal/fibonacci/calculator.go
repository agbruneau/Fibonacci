// @module(fibonacci)
// @author(Jules)
// @date(2023-10-27)
// @version(1.1)
//
// @description(Ce module est le cœur architectural du calculateur, définissant les interfaces et les optimisations fondamentales.)
// @pedagogical(Illustre les patrons Décorateur et Adaptateur, l'optimisation mémoire avec `sync.Pool`, et la gestion de l'immuabilité.)
package fibonacci

import (
	"context"
	"math/big"
	"sync"
)

const (
	// @const(MaxFibUint64)
	// @description(F(93), le plus grand Fibonacci stockable dans un `uint64`.)
	// @rationale(Limite pour l'optimisation O(1) via la table de consultation (LUT).)
	MaxFibUint64 = 93

	// @const(DefaultParallelThreshold)
	// @description(Seuil (en bits) pour activer la multiplication parallèle.)
	// @rationale(En deçà, le coût de synchronisation des goroutines excède le gain en performance.)
	DefaultParallelThreshold = 2048
)

// @struct(ProgressUpdate)
// @description(DTO pour la communication de la progression du calcul.)
type ProgressUpdate struct {
	CalculatorIndex int     // Identifiant du calculateur.
	Value           float64 // Progression normalisée (0.0 à 1.0).
}

// @type(ProgressReporter)
// @description(Callback fonctionnel pour rapporter la progression.)
// @pedagogical(Découple les algorithmes du mécanisme de communication (canaux), respectant le principe d'inversion de dépendances.)
type ProgressReporter func(progress float64)

// @interface(Calculator)
// @description(Interface publique du module, point d'entrée pour l'orchestrateur.)
type Calculator interface {
	Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error)
	Name() string
}

// @interface(coreCalculator)
// @description(Interface interne pour un algorithme de calcul pur.)
// @pedagogical(Exemple du Principe de Ségrégation des Interfaces (SOLID) : contrat simple, sans dépendances d'orchestration.)
type coreCalculator interface {
	CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error)
	Name() string
}

// @struct(FibCalculator)
// @description(Implémentation de `Calculator` appliquant les patrons Décorateur et Adaptateur.)
// @pattern(Decorator, Adapter)
type FibCalculator struct {
	core coreCalculator
}

// @function(NewCalculator)
// @description(Factory pour construire le décorateur `FibCalculator` autour d'un `coreCalculator`.)
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: `coreCalculator` ne peut être nil")
	}
	return &FibCalculator{core: core}
}

// @method(Name)
// @description(Délègue l'appel à l'objet `coreCalculator` enveloppé.)
func (c *FibCalculator) Name() string {
	return c.core.Name()
}

// @method(Calculate)
// @description(Orchestre les rôles de décorateur et d'adaptateur.)
// @architecture(
//   1. Adapte le `chan<- ProgressUpdate` en un `ProgressReporter` simple.
//   2. Décore le calcul avec une optimisation "fast path" (LUT).
//   3. Délègue au `coreCalculator`.
//   4. Assure que la progression finale est toujours rapportée.
// )
func (c *FibCalculator) Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error) {
	reporter := func(progress float64) {
		if progressChan == nil {
			return
		}
		if progress > 1.0 {
			progress = 1.0
		}
		update := ProgressUpdate{CalculatorIndex: calcIndex, Value: progress}
		select {
		case progressChan <- update: // Envoi non-bloquant
		default: // Abandon si le canal est plein
		}
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return lookupSmall(n), nil
	}

	result, err := c.core.CalculateCore(ctx, reporter, n, threshold, fftThreshold)
	if err == nil && result != nil {
		reporter(1.0) // Filet de sécurité pour garantir 100%
	}
	return result, err
}

// @variable(fibLookupTable)
// @description(Table de consultation (LUT) pour les petites valeurs de Fibonacci.)
var fibLookupTable [MaxFibUint64 + 1]*big.Int

// @function(init)
// @description(Pré-calcule la LUT au démarrage du programme.)
func init() {
	fibLookupTable[0] = big.NewInt(0)
	if MaxFibUint64 > 0 {
		fibLookupTable[1] = big.NewInt(1)
		for i := uint64(2); i <= MaxFibUint64; i++ {
			fibLookupTable[i] = new(big.Int).Add(fibLookupTable[i-1], fibLookupTable[i-2])
		}
	}
}

// @function(lookupSmall)
// @description(Récupère une valeur de la LUT de manière immuable.)
// @pedagogical(Retourne une NOUVELLE copie pour garantir l'immuabilité de la table partagée, prévenant des effets de bord.)
func lookupSmall(n uint64) *big.Int {
	return new(big.Int).Set(fibLookupTable[n])
}

// @section(Object Pooling)
// @description(Infrastructure pour la réutilisation d'objets via `sync.Pool` afin de minimiser les allocations et la pression sur le GC.)

// @struct(Pool[T])
// @description(Wrapper générique et typé pour `sync.Pool`.)
// @pedagogical(Utilise les génériques Go pour la sécurité de type, évitant les assertions de type `interface{}`.)
type Pool[T any] struct {
	pool sync.Pool
}

func NewPool[T any](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{New: func() any { return newFunc() }},
	}
}
func (p *Pool[T]) Get() T  { return p.pool.Get().(T) }
func (p *Pool[T]) Put(x T) { p.pool.Put(x) }

// @interface(Resettable)
// @description(Contrat pour les objets poolables qui nécessitent une réinitialisation.)
type Resettable interface {
	Reset()
}

// @function(acquireFromPool)
// @description(Récupère un objet du pool et le réinitialise.)
// @pedagogical(La réinitialisation est cruciale pour éviter la corruption de données avec des objets "sales".)
func acquireFromPool[T Resettable](p *Pool[T]) T {
	item := p.Get()
	item.Reset()
	return item
}

func releaseToPool[T any](p *Pool[T], item T) {
	if any(item) == nil {
		return
	}
	p.Put(item)
}

// @pool(bigIntPool)
// @description(Pool pour les objets `*big.Int`.)
type ResettableBigInt struct{ *big.Int }

func (b *ResettableBigInt) Reset() { b.SetInt64(0) }

var bigIntPool = NewPool(func() *ResettableBigInt {
	return &ResettableBigInt{new(big.Int)}
})

func acquireBigInt() *ResettableBigInt { return acquireFromPool(bigIntPool) }
func releaseBigInt(b *ResettableBigInt) { releaseToPool(bigIntPool, b) }

// @struct(calculationState)
// @description(Regroupe les variables temporaires pour l'algorithme Fast Doubling.)
type calculationState struct {
	f_k, f_k1, t1, t2, t3, t4 *big.Int
}

func (s *calculationState) Reset() {
	s.f_k.SetInt64(0)
	s.f_k1.SetInt64(1)
	// Les temporaires (t1-t4) sont écrasés avant lecture, pas besoin de reset.
}

var statePool = NewPool(func() *calculationState {
	return &calculationState{
		f_k: new(big.Int), f_k1: new(big.Int), t1: new(big.Int),
		t2: new(big.Int), t3: new(big.Int), t4: new(big.Int),
	}
})

func acquireState() *calculationState { return acquireFromPool(statePool) }
func releaseState(s *calculationState) { releaseToPool(statePool, s) }

// @struct(matrix)
// @description(Représente une matrice 2x2 de `*big.Int`.)
type matrix struct{ a, b, c, d *big.Int }

func newMatrix() *matrix {
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
}
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// @struct(matrixState)
// @description(Regroupe les variables pour l'exponentiation matricielle.)
type matrixState struct {
	res, p, tempMatrix             *matrix
	t1, t2, t3, t4, t5, t6, t7, t8 *big.Int
}

func (s *matrixState) Reset() {
	s.res.SetIdentity()
	s.p.SetBaseQ()
}

var matrixStatePool = NewPool(func() *matrixState {
	return &matrixState{
		res: newMatrix(), p: newMatrix(), tempMatrix: newMatrix(),
		t1: new(big.Int), t2: new(big.Int), t3: new(big.Int), t4: new(big.Int),
		t5: new(big.Int), t6: new(big.Int), t7: new(big.Int), t8: new(big.Int),
	}
})

func acquireMatrixState() *matrixState { return acquireFromPool(matrixStatePool) }
func releaseMatrixState(s *matrixState) { releaseToPool(matrixStatePool, s) }