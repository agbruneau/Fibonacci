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
// scientifique avec l'exposant rendu en caractères Unicode superscript, facilitant
// ainsi la lecture de nombres très volumineux.
//
// Techniques employées :
// - Algorithme du doublement pour calculer Fibonacci(n) efficacement.
// - Parallélisation des multiplications (big.Int) par goroutines pour exploiter
//   la puissance des processeurs multi‑cœurs.
// - Utilisation d’un contexte avec timeout pour la robustesse du calcul.
// - Formatage en notation scientifique avec conversion des exposants en superscript.
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
	highest := 0
	for i := 31; i >= 0; i-- {
		if n&(1<<i) != 0 {
			highest = i
			break
		}
	}

	// Parcours des bits de n, du plus significatif au moins significatif.
	for i := highest; i >= 0; i-- {
		// Calcul de deuxB = 2 * b (opération rapide via un décalage de bits).
		twoB := new(big.Int).Lsh(b, 1)
		// Calcul de temp = 2*b - a.
		temp := new(big.Int).Sub(twoB, a)

		// Création de canaux tamponnés pour récupérer les résultats des multiplications.
		cChan := make(chan *big.Int, 1)
		t1Chan := make(chan *big.Int, 1)
		t2Chan := make(chan *big.Int, 1)

		// Lancement de la goroutine pour calculer c = a * (2*b - a).
		go func(a, temp *big.Int) {
			cChan <- new(big.Int).Mul(a, temp)
		}(new(big.Int).Set(a), temp)

		// Lancement de la goroutine pour calculer t1 = a * a.
		go func(a *big.Int) {
			t1Chan <- new(big.Int).Mul(a, a)
		}(new(big.Int).Set(a))

		// Lancement de la goroutine pour calculer t2 = b * b.
		go func(b *big.Int) {
			t2Chan <- new(big.Int).Mul(b, b)
		}(new(big.Int).Set(b))

		// Récupération des résultats des multiplications via les canaux.
		c := <-cChan
		t1 := <-t1Chan
		t2 := <-t2Chan

		// Calcul de d = a*a + b*b.
		d := new(big.Int).Add(t1, t2)

		// Mise à jour des valeurs (a, b) selon la valeur du bit courant de n.
		if n&(1<<uint(i)) != 0 {
			a.Set(d)
			b.Add(c, d)
		} else {
			a.Set(c)
			b.Set(d)
		}
	}
	// À la fin de la boucle, a contient Fibonacci(n).
	return a, nil
}

// toSuperscript convertit une chaîne de chiffres (et éventuellement le signe '-') en
// leurs équivalents en exposants Unicode. Cette fonction est utilisée pour afficher
// l'exposant en notation scientifique de manière lisible.
func toSuperscript(s string) string {
	// Définition d'une table de correspondance entre chiffres et leurs équivalents en
	// caractères superscript.
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
// rendu en caractères Unicode superscript.
// Par exemple, un nombre tel que 354224848179261915075 sera formaté en "3.54224×10²⁰".
func formatBigIntSup(n *big.Int) string {
	s := n.String()
	// Si le nombre contient un seul chiffre, il est retourné tel quel.
	if len(s) <= 1 {
		return s
	}
	// Détermination du nombre de chiffres significatifs (ici 6 chiffres au total).
	var significand string
	if len(s) > 6 {
		significand = s[:1] + "." + s[1:6]
	} else {
		significand = s[:1] + "." + s[1:]
	}
	// Calcul de l'exposant : le nombre de chiffres moins 1.
	exponent := len(s) - 1
	// Conversion de l'exposant en notation superscript.
	supExp := toSuperscript(fmt.Sprintf("%d", exponent))
	return fmt.Sprintf("%s×10%s", significand, supExp)
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

	// Formatage du résultat en notation scientifique avec l'exposant en caractères superscript.
	formattedResult := formatBigIntSup(fibResult)
	fmt.Printf("\nRésultat :\n")
	fmt.Printf("  Fibonacci(%d) : %s\n", config.M, formattedResult)
}
