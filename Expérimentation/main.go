// =============================================================================
// Programme : Calcul et affichage de Fibonacci(n) en notation scientifique
//
// Description :
// Ce programme calcule le n‑ième nombre de Fibonacci en utilisant l'algorithme
// du doublement (doubling method), qui permet de réduire la complexité du calcul
// à O(log n). Pour optimiser les opérations sur de grands entiers (big.Int), le
// code parallélise les multiplications coûteuses à l'aide de goroutines et de canaux.
// Un contexte avec timeout est mis en place pour limiter la durée d'exécution, et
// des métriques de performance sont collectées. Le résultat est affiché en notation
// scientifique avec l'exposant rendu en notation exponentielle (ex: 1.23e45), facilitant
// ainsi la lecture de nombres très volumineux.
//
// Techniques employées :
// - Algorithme du doublement pour calculer Fibonacci(n) efficacement.
// - Parallélisation des multiplications (big.Int) par goroutines pour exploiter
//   la puissance des processeurs multicœurs.
// - Utilisation d’un contexte avec timeout pour la robustesse du calcul.
// - Formatage en notation scientifique avec notation exponentielle.
// =============================================================================

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync/atomic"
	"time"
)

// Configuration centralise les paramètres configurables du programme.
type Configuration struct {
	M       int           // Calcul de Fibonacci(M)
	Timeout time.Duration // Durée maximale d'exécution
}

// DefaultConfig retourne une configuration par défaut.
// Par défaut, la valeur de M est fixée à 100 (modifiable selon les besoins),
// et le timeout est défini à 5 minutes.
func DefaultConfig() Configuration {
	return Configuration{
		M:       200000000,
		Timeout: 5 * time.Minute,
	}
}

// Metrics conserve des informations de performance du calcul.
type Metrics struct {
	StartTime         time.Time // Heure de début du calcul
	EndTime           time.Time // Heure de fin du calcul
	TotalCalculations int64     // Nombre de calculs réalisés (pour le moment, 1 calcul)
}

// NewMetrics initialise les métriques en enregistrant l'heure de début.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddCalculations incrémente de manière atomique le compteur de calculs.
func (m *Metrics) AddCalculations(n int64) {
	atomic.AddInt64(&m.TotalCalculations, n)
}

// FibCalculator encapsule le calcul du n‑ième nombre de Fibonacci.
type FibCalculator struct{}

// NewFibCalculator retourne une nouvelle instance de FibCalculator.
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{}
}

// Calculate retourne Fibonacci(n) pour n ≥ 0.
// Les cas particuliers n=0 et n=1 sont traités directement.
func (fc *FibCalculator) Calculate(n int) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("n doit être non négatif")
	}
	if n == 0 {
		return big.NewInt(0), nil
	}
	if n == 1 {
		return big.NewInt(1), nil
	}
	// Pour n supérieur à 1, on utilise l'algorithme du doublement parallélisé.
	return fibDoublingParallel(n)
}

// multiplicationResult représente le résultat d'une opération de multiplication.
type multiplicationResult struct {
	result *big.Int
	err    error
}

// fibDoublingParallel calcule Fibonacci(n) en utilisant l'algorithme itératif
// du doublement (doubling method) avec parallélisation des multiplications lourdes.
// L'algorithme parcourt les bits de n, du bit le plus significatif au moins significatif,
// et à chaque itération, lance des goroutines pour calculer simultanément :
// - c = a * (2*b - a)
// - t1 = a * a
// - t2 = b * b
// Puis, en fonction du bit courant de n, les valeurs de a et b sont mises à jour.
func fibDoublingParallel(n int) (*big.Int, error) {
	// Initialisation : a = F(0) = 0, b = F(1) = 1
	a := big.NewInt(0)
	b := big.NewInt(1)

	// Détermination du bit le plus significatif de n.
	highest := determineHighestBit(n)
	// Création des big int utilisé dans la boucle
	twoB := new(big.Int)
	temp := new(big.Int)
	c := new(big.Int)
	d := new(big.Int)

	// Parcours des bits de n, du plus significatif au moins significatif.
	for i := highest; i >= 0; i-- {
		// Calcul de deuxB = 2 * b (opération rapide via un décalage de bits).
		twoB.Lsh(b, 1)
		// Calcul de temp = 2*b - a.
		temp.Sub(twoB, a)

		// Création et configuration des canaux pour les résultats des multiplications.
		cChan, t1Chan, t2Chan, errChan := setupMultiplicationChannels()

		// Lancement des goroutines pour effectuer les multiplications.
		launchMultiplicationGoroutines(a, temp, b, cChan, t1Chan, t2Chan, errChan)

		// Récupération des résultats des multiplications et gestion des erreurs.
		if err := handleMultiplicationResults(cChan, t1Chan, t2Chan, errChan, c, d); err != nil {
			return nil, err
		}

		// Mise à jour des valeurs (a, b) selon la valeur du bit courant de n.
		updateFibonacciValues(n, i, a, b, c, d)
	}
	// À la fin de la boucle, a contient Fibonacci(n).
	return a, nil
}

// determineHighestBit Détermine le bit le plus significatif de n.
func determineHighestBit(n int) int {
	highest := 0
	for i := 31; i >= 0; i-- {
		if n&(1<<i) != 0 {
			highest = i
			break
		}
	}
	return highest
}

// setupMultiplicationChannels crée et retourne les canaux pour les résultats des multiplications.
func setupMultiplicationChannels() (cChan, t1Chan, t2Chan chan multiplicationResult, errChan chan error) {
	cChan = make(chan multiplicationResult, 1)
	t1Chan = make(chan multiplicationResult, 1)
	t2Chan = make(chan multiplicationResult, 1)
	errChan = make(chan error, 3)
	return
}

// launchMultiplicationGoroutines lance les goroutines pour effectuer les multiplications.
func launchMultiplicationGoroutines(a, temp, b *big.Int, cChan, t1Chan, t2Chan chan multiplicationResult, errChan chan error) {
	go multiply(a, temp, cChan, errChan)
	go multiply(a, a, t1Chan, errChan)
	go multiply(b, b, t2Chan, errChan)
}

// multiply effectue la multiplication de deux grands entiers et envoie le résultat sur le canal.
func multiply(x, y *big.Int, resultChan chan multiplicationResult, errChan chan error) {
	result, err := performMultiplication(x, y)
	if err != nil {
		errChan <- err
		return
	}
	resultChan <- multiplicationResult{result: result, err: nil}
}

// performMultiplication effectue la multiplication et retourne le résultat.
func performMultiplication(x, y *big.Int) (*big.Int, error) {
	if x == nil || y == nil {
		return nil, errors.New("cannot multiply nil big.Int")
	}
	return new(big.Int).Mul(x, y), nil
}

// handleMultiplicationResults récupère les résultats des multiplications et gère les erreurs.
func handleMultiplicationResults(cChan, t1Chan, t2Chan chan multiplicationResult, errChan chan error, c, d *big.Int) error {
	var err error
	cResult := <-cChan
	t1Result := <-t1Chan
	t2Result := <-t2Chan

	select {
	case err = <-errChan:
		return err
	default:
	}
	if cResult.err != nil {
		return cResult.err
	}
	if t1Result.err != nil {
		return t1Result.err
	}
	if t2Result.err != nil {
		return t2Result.err
	}
	c.Set(cResult.result)
	d.Add(t1Result.result, t2Result.result)
	return nil
}

// updateFibonacciValues met à jour les valeurs de a et b selon la valeur du bit courant de n.
func updateFibonacciValues(n, i int, a, b, c, d *big.Int) {
	if n&(1<<uint(i)) != 0 {
		a.Set(d)
		b.Add(c, d)
	} else {
		a.Set(c)
		b.Set(d)
	}
}

// formatBigIntExp formate un grand entier en notation scientifique avec l'exposant
// en notation exponentielle (ex: 1.23e45).
func formatBigIntExp(n *big.Int) string {
	if n.Sign() == 0 {
		return "0"
	}

	s := n.String()
	isNegative := false
	if s[0] == '-' {
		isNegative = true
		s = s[1:]
	}

	// Cas simple si le nombre a un seul chiffre
	if len(s) <= 1 {
		if isNegative {
			return "-" + s
		}
		return s
	}

	// Détermination du nombre de chiffres significatifs (ici, 6 chiffres).
	var significand string
	if len(s) > 6 {
		significand = s[:1] + "." + s[1:6]
	} else {
		significand = s[:1] + "." + s[1:] + string(make([]byte, 6-len(s)))
		for i := 0; i < 6-len(s); i++ {
			significand += "0"
		}
	}

	// Calcul de l'exposant.
	exponent := len(s) - 1

	// Formatage de la sortie.
	if isNegative {
		return fmt.Sprintf("-%se%d", significand, exponent)
	}
	return fmt.Sprintf("%se%d", significand, exponent)
}

// main constitue le point d'entrée du programme.
// Il configure le runtime, initialise les paramètres et les métriques, et déclenche le calcul.
func main() {
	// Configuration explicite pour exploiter tous les cœurs disponibles.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Récupération de la configuration par défaut.
	config := DefaultConfig()

	// Initialisation des métriques pour mesurer la performance.
	metrics := NewMetrics()

	// Création d'un contexte avec timeout pour limiter la durée d'exécution du calcul.
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Instanciation du calculateur de Fibonacci.
	fc := NewFibCalculator()

	// Création de canaux pour récupérer le résultat ou une éventuelle erreur.
	resultChan := make(chan *big.Int, 1)
	errorChan := make(chan error, 1)

	// Lancement du calcul de Fibonacci dans une goroutine.
	go func() {
		fib, err := fc.Calculate(config.M)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- fib
	}()

	// Sélection entre le résultat, une erreur ou un timeout.
	var fibResult *big.Int
	select {
	case <-ctx.Done():
		log.Fatalf("Délai d'exécution dépassé : %v", ctx.Err())
	case err := <-errorChan:
		log.Fatalf("Erreur lors du calcul de Fibonacci : %v", err)
	case fibResult = <-resultChan:
		// Le calcul s'est terminé correctement.
	}

	// Comptabilisation du calcul effectué.
	metrics.AddCalculations(1)
	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)

	// Calcul du temps moyen par calcul.
	var avgTime time.Duration
	if metrics.TotalCalculations > 0 {
		avgTime = duration / time.Duration(metrics.TotalCalculations)
	}

	// Affichage de la configuration et des informations de performance.
	fmt.Printf("\nConfiguration :\n")
	fmt.Printf("  Valeur de M             : %d\n", config.M)
	fmt.Printf("  Timeout                 : %v\n", config.Timeout)
	fmt.Printf("  Nombre de cœurs utilisés: %d\n", runtime.NumCPU())

	fmt.Printf("\nPerformance :\n")
	fmt.Printf("  Temps total d'exécution : %v\n", duration)
	fmt.Printf("  Nombre de calculs       : %d\n", metrics.TotalCalculations)
	fmt.Printf("  Temps moyen par calcul  : %v\n", avgTime)

	// Formatage du résultat en notation scientifique avec l'exposant en notation exponentielle.
	formattedResult := formatBigIntExp(fibResult)
	fmt.Printf("\nRésultat :\n")
	fmt.Printf("  Fibonacci(%d) : %s\n", config.M, formattedResult)
}
