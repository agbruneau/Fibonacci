// Le paquetage fibonacci fournit des implémentations pour le calcul des nombres de
// la suite de Fibonacci. Il expose une interface `Calculator` qui abstrait
// l'algorithme de calcul sous-jacent, permettant ainsi d'utiliser différentes
// stratégies (par exemple, Fast Doubling, Exponentiation Matricielle) de manière
// interchangeable. Le paquetage intègre également des optimisations telles
// qu'une table de consultation (LUT) pour les petites valeurs et une gestion
// de la mémoire via des pools d'objets pour minimiser la pression sur le
// ramasse-miettes (GC).
package fibonacci

import (
	"context"
	"math/big"
	"sync"
)

const (
	// MaxFibUint64 représente l'indice du plus grand nombre de Fibonacci
	// calculable sur un entier non signé de 64 bits.
	MaxFibUint64 = 93

	// DefaultParallelThreshold définit le seuil en bits à partir duquel les
	// multiplications de grands entiers sont parallélisées.
	DefaultParallelThreshold = 4096
)

// ProgressUpdate est un objet de transfert de données (DTO) qui encapsule
// l'état de progression d'un calcul.
type ProgressUpdate struct {
	CalculatorIndex int     // Identifiant unique du calculateur.
	Value           float64 // Valeur normalisée de la progression [0.0, 1.0].
}

// ProgressReporter définit le type fonctionnel pour un callback de rapport de
// progression.
type ProgressReporter func(progress float64)

// Calculator définit l'interface publique pour un calculateur Fibonacci.
type Calculator interface {
	// Calculate exécute le calcul du n-ième nombre de Fibonacci.
	Calculate(ctx context.Context, progressChan chan<- ProgressUpdate, calcIndex int, n uint64, threshold int, fftThreshold int) (*big.Int, error)
	// Name retourne le nom de l'algorithme de calcul.
	Name() string
}

// coreCalculator définit l'interface interne pour un algorithme de calcul pur.
type coreCalculator interface {
	CalculateCore(ctx context.Context, reporter ProgressReporter, n uint64, threshold int, fftThreshold int) (*big.Int, error)
	Name() string
}

// FibCalculator est une implémentation de l'interface `Calculator` qui utilise
// le patron de conception Décorateur pour ajouter des fonctionnalités autour
// d'un `coreCalculator`.
type FibCalculator struct {
	core coreCalculator
}

// NewCalculator est une fonction de fabrique qui construit un `FibCalculator`.
func NewCalculator(core coreCalculator) Calculator {
	if core == nil {
		panic("fibonacci: l'implémentation de `coreCalculator` ne peut être nulle")
	}
	return &FibCalculator{core: core}
}

// Name retourne le nom du calculateur encapsulé.
func (c *FibCalculator) Name() string {
	return c.core.Name()
}

// Calculate orchestre le calcul. Il adapte le canal de progression en un
// simple `ProgressReporter`, applique une optimisation pour les petites valeurs
// de `n` et délègue le calcul principal au `coreCalculator`.
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
		case progressChan <- update:
		default:
		}
	}

	if n <= MaxFibUint64 {
		reporter(1.0)
		return lookupSmall(n), nil
	}

	result, err := c.core.CalculateCore(ctx, reporter, n, threshold, fftThreshold)
	if err == nil && result != nil {
		reporter(1.0)
	}
	return result, err
}

var fibLookupTable [MaxFibUint64 + 1]*big.Int

func init() {
	fibLookupTable[0] = big.NewInt(0)
	if MaxFibUint64 > 0 {
		fibLookupTable[1] = big.NewInt(1)
		for i := uint64(2); i <= MaxFibUint64; i++ {
			fibLookupTable[i] = new(big.Int).Add(fibLookupTable[i-1], fibLookupTable[i-2])
		}
	}
}

// lookupSmall retourne une copie du n-ième nombre de Fibonacci à partir de la
// table de consultation, garantissant l'immuabilité de la table.
func lookupSmall(n uint64) *big.Int {
	return new(big.Int).Set(fibLookupTable[n])
}

// calculationState agrège les variables temporaires pour l'algorithme
// "Fast Doubling", permettant une gestion efficace via un pool d'objets.
type calculationState struct {
	f_k, f_k1, t1, t2, t3, t4 *big.Int
}

// Reset réinitialise l'état pour une nouvelle utilisation.
func (s *calculationState) Reset() {
	s.f_k.SetInt64(0)
	s.f_k1.SetInt64(1)
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &calculationState{
			f_k:  new(big.Int),
			f_k1: new(big.Int),
			t1:   new(big.Int),
			t2:   new(big.Int),
			t3:   new(big.Int),
			t4:   new(big.Int),
		}
	},
}

// acquireState obtient un état du pool et le réinitialise.
func acquireState() *calculationState {
	s := statePool.Get().(*calculationState)
	s.Reset()
	return s
}

// releaseState remet un état dans le pool.
func releaseState(s *calculationState) {
	statePool.Put(s)
}

// matrix représente une matrice 2x2 de `*big.Int`.
type matrix struct{ a, b, c, d *big.Int }

// newMatrix alloue une nouvelle matrice.
func newMatrix() *matrix {
	return &matrix{new(big.Int), new(big.Int), new(big.Int), new(big.Int)}
}

// Set copie les valeurs d'une autre matrice.
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// SetIdentity configure la matrice en tant que matrice identité.
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// SetBaseQ configure la matrice avec la matrice de base de Fibonacci.
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// matrixState agrège les variables pour l'algorithme d'exponentiation
// matricielle.
type matrixState struct {
	res, p, tempMatrix             *matrix
	t1, t2, t3, t4, t5, t6, t7, t8 *big.Int
}

// Reset réinitialise l'état pour une nouvelle utilisation.
func (s *matrixState) Reset() {
	s.res.SetIdentity()
	s.p.SetBaseQ()
}

var matrixStatePool = sync.Pool{
	New: func() interface{} {
		return &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			t1: new(big.Int), t2: new(big.Int), t3: new(big.Int), t4: new(big.Int),
			t5: new(big.Int), t6: new(big.Int), t7: new(big.Int), t8: new(big.Int),
		}
	},
}

// acquireMatrixState obtient un état du pool et le réinitialise.
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)
	s.Reset()
	return s
}

// releaseMatrixState remet un état dans le pool.
func releaseMatrixState(s *matrixState) {
	matrixStatePool.Put(s)
}