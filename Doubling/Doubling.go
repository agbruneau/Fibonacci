// =============================================================================
// Programme : Calcul et affichage de Fibonacci(n) en notation scientifique
// Auteur    : [Votre nom ou pseudonyme]
// Date      : [Date de création/modification]
// Version   : 2.0
//
// Description :
// Ce programme implémente le calcul du n-ième nombre de Fibonacci en utilisant
// l'algorithme du doublement (doubling method) optimisé par la parallélisation
// des opérations sur de grands entiers (big.Int) à l'aide de goroutines.
// L’algorithme présente une complexité en O(log n) et exploite pleinement la
// puissance de calcul disponible (machine multi‑cœurs) en configurant explicitement
// runtime.GOMAXPROCS. Le résultat est affiché en notation scientifique avec un
// exposant en caractères Unicode superscript.
// =============================================================================

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync/atomic"
	"time"
)

// Configuration centralise les paramètres configurables.
type Configuration struct {
	M       int           // Calcul de Fibonacci(M)
	Timeout time.Duration // Durée maximale d'exécution
}

// DefaultConfig retourne une configuration par défaut.
func DefaultConfig() Configuration {
	return Configuration{
		// Par défaut, on calcule Fibonacci(100) (modifiable selon les besoins)
		M:       100000000,
		Timeout: 5 * time.Minute, // Timeout de 5 minutes
	}
}

// Metrics conserve quelques métriques de performance.
type Metrics struct {
	StartTime         time.Time // Heure de début
	EndTime           time.Time // Heure de fin
	TotalCalculations int64     // Nombre de calculs réalisés
}

// NewMetrics initialise les métriques avec l'heure de début.
func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

// AddCalculations incrémente le compteur via une opération atomique.
func (m *Metrics) AddCalculations(n int64) {
	atomic.AddInt64(&m.TotalCalculations, n)
}

// FibCalculator encapsule le calcul du n-ième nombre de Fibonacci.
type FibCalculator struct{}

// NewFibCalculator retourne une nouvelle instance de FibCalculator.
func NewFibCalculator() *FibCalculator {
	return &FibCalculator{}
}

// Calculate retourne F(n) pour n ≥ 0.
// Pour n = 0 ou 1, le résultat est retourné directement.
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
	return fibDoublingParallel(n)
}

// fibDoublingParallel calcule F(n) en utilisant l'algorithme itératif du doublement
// avec parallélisation des opérations coûteuses. L'algorithme parcourt les bits de n
// du plus significatif au moins significatif et, pour chaque itération, lance des
// goroutines pour calculer simultanément les multiplications.
func fibDoublingParallel(n int) (*big.Int, error) {
	// Initialisation : a = F(0) = 0, b = F(1) = 1
	a := big.NewInt(0)
	b := big.NewInt(1)

	// Détermination du bit le plus significatif de n
	highest := 0
	for i := 31; i >= 0; i-- {
		if n&(1<<i) != 0 {
			highest = i
			break
		}
	}

	// Parcours des bits de n, du plus significatif au moins significatif
	for i := highest; i >= 0; i-- {
		// Calcul de deuxB = 2 * b
		twoB := new(big.Int).Lsh(b, 1)
		// Calcul de temp = 2*b - a
		temp := new(big.Int).Sub(twoB, a)

		// Création de canaux pour récupérer les résultats des multiplications
		cChan := make(chan *big.Int, 1)
		t1Chan := make(chan *big.Int, 1)
		t2Chan := make(chan *big.Int, 1)

		// Calcul de c = a * (2*b - a) en parallèle
		go func(a, temp *big.Int) {
			cChan <- new(big.Int).Mul(a, temp)
		}(new(big.Int).Set(a), temp)

		// Calcul de t1 = a * a en parallèle
		go func(a *big.Int) {
			t1Chan <- new(big.Int).Mul(a, a)
		}(new(big.Int).Set(a))

		// Calcul de t2 = b * b en parallèle
		go func(b *big.Int) {
			t2Chan <- new(big.Int).Mul(b, b)
		}(new(big.Int).Set(b))

		// Récupération des résultats
		c := <-cChan
		t1 := <-t1Chan
		t2 := <-t2Chan

		// Calcul de d = a*a + b*b
		d := new(big.Int).Add(t1, t2)

		// Mise à jour de (a, b) selon le bit courant de n
		if n&(1<<uint(i)) != 0 {
			a.Set(d)
			b.Add(c, d)
		} else {
			a.Set(c)
			b.Set(d)
		}
	}
	return a, nil
}

// toSuperscript convertit une chaîne composée de chiffres (et éventuellement le signe '-')
// en leurs équivalents en exposants Unicode.
func toSuperscript(s string) string {
	supDigits := map[rune]rune{
		'0': '⁰',
		'1': '¹',
		'2': '²',
		'3': '³',
		'4': '⁴',
		'5': '⁵',
		'6': '⁶',
		'7': '⁷',
		'8': '⁸',
		'9': '⁹',
		'-': '⁻',
	}
	result := ""
	for _, r := range s {
		if sup, ok := supDigits[r]; ok {
			result += string(sup)
		} else {
			result += string(r)
		}
	}
	return result
}

// formatBigIntSup formate un grand entier en notation scientifique avec l'exposant
// rendu en caractères Unicode superscript. Par exemple : "3.54224×10²⁰".
func formatBigIntSup(n *big.Int) string {
	s := n.String()
	if len(s) <= 1 {
		return s
	}
	// Choix du nombre de chiffres significatifs (ici 6 chiffres au total)
	var significand string
	if len(s) > 6 {
		significand = s[:1] + "." + s[1:6]
	} else {
		significand = s[:1] + "." + s[1:]
	}
	exponent := len(s) - 1
	supExp := toSuperscript(fmt.Sprintf("%d", exponent))
	return fmt.Sprintf("%s×10%s", significand, supExp)
}

func main() {
	// Configuration explicite pour exploiter tous les cœurs disponibles
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Initialisation de la configuration et des métriques.
	config := DefaultConfig()
	metrics := NewMetrics()

	// Création d'un contexte avec timeout pour limiter la durée d'exécution.
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Calcul de Fibonacci(config.M)
	fc := NewFibCalculator()
	resultChan := make(chan *big.Int, 1)
	errorChan := make(chan error, 1)

	go func() {
		fib, err := fc.Calculate(config.M)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- fib
	}()

	var fibResult *big.Int
	select {
	case <-ctx.Done():
		log.Fatalf("Délai d'exécution dépassé : %v", ctx.Err())
	case err := <-errorChan:
		log.Fatalf("Erreur lors du calcul de Fibonacci : %v", err)
	case fibResult = <-resultChan:
		// Calcul terminé.
	}

	// Comptabilisation du calcul effectué.
	metrics.AddCalculations(1)
	metrics.EndTime = time.Now()
	duration := metrics.EndTime.Sub(metrics.StartTime)
	var avgTime time.Duration
	if metrics.TotalCalculations > 0 {
		avgTime = duration / time.Duration(metrics.TotalCalculations)
	}

	// Affichage des résultats et des métriques.
	fmt.Printf("\nConfiguration :\n")
	fmt.Printf("  Valeur de M             : %d\n", config.M)
	fmt.Printf("  Timeout                 : %v\n", config.Timeout)
	fmt.Printf("  Nombre de cœurs utilisés: %d\n", runtime.NumCPU())

	fmt.Printf("\nPerformance :\n")
	fmt.Printf("  Temps total d'exécution : %v\n", duration)
	fmt.Printf("  Nombre de calculs       : %d\n", metrics.TotalCalculations)
	fmt.Printf("  Temps moyen par calcul  : %v\n", avgTime)

	// Affichage du résultat en notation scientifique avec l'exposant en superscript.
	formattedResult := formatBigIntSup(fibResult)
	fmt.Printf("\nRésultat :\n")
	fmt.Printf("  Fibonacci(%d) : %s\n", config.M, formattedResult)
}
