// Package fibonacci fournit des implémentations optimisées pour le calcul de la suite de Fibonacci.
package fibonacci

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
)

// AlgorithmKey est un identifiant unique pour chaque algorithme.
type AlgorithmKey string

// Calculator définit l'interface que toutes les implémentations doivent satisfaire.
type Calculator interface {
	// Calculate calcule F(N). Il doit respecter le contexte et utiliser le pool fourni.
	Calculate(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)
}

// Algorithm encapsule les métadonnées et l'implémentation.
type Algorithm struct {
	Key  AlgorithmKey
	Name string
	Impl Calculator
}

var registry = make(map[AlgorithmKey]Algorithm)

// register est utilisé par les fichiers d'implémentation (via init()) pour s'ajouter au registre.
func register(key AlgorithmKey, name string, impl Calculator) {
	if _, exists := registry[key]; exists {
		panic(fmt.Sprintf("Algorithme déjà enregistré: %s", key))
	}
	registry[key] = Algorithm{Key: key, Name: name, Impl: impl}
}

// Get récupère un algorithme par sa clé.
func Get(key AlgorithmKey) (Algorithm, error) {
	algo, ok := registry[key]
	if !ok {
		return Algorithm{}, fmt.Errorf("algorithme non trouvé: %s", key)
	}
	return algo, nil
}

// IsRegistered vérifie si une clé est valide.
func IsRegistered(key AlgorithmKey) bool {
	_, ok := registry[key]
	return ok
}

// ListAlgorithms retourne une liste triée de tous les algorithmes disponibles.
func ListAlgorithms() []Algorithm {
	algos := make([]Algorithm, 0, len(registry))
	for _, algo := range registry {
		algos = append(algos, algo)
	}
	sort.Slice(algos, func(i, j int) bool {
		return algos[i].Key < algos[j].Key
	})
	return algos
}

// NewIntPool crée un sync.Pool configuré pour recycler des instances de big.Int.
func NewIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} { return new(big.Int) },
	}
}

// handleBaseCases gère les cas triviaux (N<0, N=0, N=1).
func handleBaseCases(n int, progress chan<- float64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("index négatif non supporté: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			// Utilisation de select/default pour ne pas bloquer si le canal est saturé.
			select {
			case progress <- 100.0:
			default:
			}
		}
		// Retourne une nouvelle instance car le résultat sera utilisé par l'appelant.
		return big.NewInt(int64(n)), nil
	}
	return nil, nil
}
