// Le paquetage cli fournit des fonctions pour construire une interface utilisateur
// en ligne de commande (CLI) pour l'application de calcul Fibonacci. Il gère
// l'affichage de la progression des calculs de manière asynchrone et formate
// les résultats pour une présentation claire et lisible.
package cli

import (
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

const (
	// ProgressRefreshRate définit la fréquence de rafraîchissement de la barre de progression.
	ProgressRefreshRate = 100 * time.Millisecond
	// ProgressBarWidth définit la largeur en caractères de la barre de progression.
	ProgressBarWidth = 40
	// TruncationLimit est le seuil de chiffres à partir duquel un résultat est tronqué.
	TruncationLimit = 100
	// DisplayEdges spécifie le nombre de chiffres à afficher au début et à la fin
	// d'un nombre tronqué.
	DisplayEdges = 25
)

// ProgressState encapsule l'état agrégé de la progression des calculs.
type ProgressState struct {
	progresses     []float64
	numCalculators int
	out            io.Writer
}

// NewProgressState initialise un nouvel état de progression.
func NewProgressState(numCalculators int, out io.Writer) *ProgressState {
	return &ProgressState{
		progresses:     make([]float64, numCalculators),
		numCalculators: numCalculators,
		out:            out,
	}
}

// Update met à jour la progression pour un calculateur spécifique.
func (ps *ProgressState) Update(index int, value float64) {
	if index >= 0 && index < len(ps.progresses) {
		ps.progresses[index] = value
	}
}

// CalculateAverage calcule la progression moyenne de tous les calculateurs.
func (ps *ProgressState) CalculateAverage() float64 {
	var totalProgress float64
	for _, p := range ps.progresses {
		totalProgress += p
	}
	if ps.numCalculators == 0 {
		return 0.0
	}
	return totalProgress / float64(ps.numCalculators)
}

// PrintBar affiche la barre de progression formatée.
func (ps *ProgressState) PrintBar(final bool) {
	avgProgress := ps.CalculateAverage()
	label := "Progression"
	if ps.numCalculators > 1 {
		label = "Progression Moyenne"
	}
	bar := progressBar(avgProgress, ProgressBarWidth)
	fmt.Fprintf(ps.out, "\r\033[K%s : %6.2f%% [%s]", label, avgProgress*100, bar)
	if final {
		fmt.Fprintln(ps.out)
	}
}

// DisplayAggregateProgress gère l'affichage de la progression de manière asynchrone.
// Elle s'exécute dans une goroutine et consomme les mises à jour de progression
// d'un canal pour rafraîchir l'interface utilisateur à intervalles réguliers.
func DisplayAggregateProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, out io.Writer) {
	defer wg.Done()
	if numCalculators <= 0 {
		for range progressChan {
		}
		return
	}

	state := NewProgressState(numCalculators, out)
	ticker := time.NewTicker(ProgressRefreshRate)
	defer ticker.Stop()

	for {
		select {
		case update, ok := <-progressChan:
			if !ok {
				state.PrintBar(true)
				return
			}
			state.Update(update.CalculatorIndex, update.Value)
		case <-ticker.C:
			state.PrintBar(false)
		}
	}
}

// progressBar génère une chaîne de caractères représentant une barre de progression.
func progressBar(progress float64, length int) string {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}
	count := int(progress * float64(length))
	var builder strings.Builder
	builder.Grow(length * 3)
	for i := 0; i < length; i++ {
		if i < count {
			builder.WriteRune('█')
		} else {
			builder.WriteRune('░')
		}
	}
	return builder.String()
}

// DisplayResult formate et affiche le résultat final du calcul.
func DisplayResult(result *big.Int, n uint64, duration time.Duration, verbose, details bool, out io.Writer) {
	bitLen := result.BitLen()
	fmt.Fprintf(out, "Taille Binaire du Résultat : %s bits.\n", formatNumberString(fmt.Sprintf("%d", bitLen)))

	if !details {
		fmt.Fprintln(out, "(Utilisez l'option -d ou --details pour un rapport complet)")
		return
	}

	fmt.Fprintln(out, "\n--- Analyse Détaillée du Résultat ---")
	if duration > 0 {
		fmt.Fprintf(out, "Temps de calcul       : %s\n", duration)
	}

	resultStr := result.String()
	numDigits := len(resultStr)
	fmt.Fprintf(out, "Nombre de chiffres    : %s\n", formatNumberString(fmt.Sprintf("%d", numDigits)))

	if numDigits > 6 {
		f := new(big.Float).SetInt(result)
		fmt.Fprintf(out, "Notation scientifique : %.6e\n", f)
	}

	fmt.Fprintln(out, "\n--- Valeur Calculée ---")
	if verbose {
		fmt.Fprintf(out, "F(%d) =\n%s\n", n, formatNumberString(resultStr))
	} else if numDigits > TruncationLimit {
		fmt.Fprintf(out, "F(%d) (tronqué) = %s...%s\n", n, resultStr[:DisplayEdges], resultStr[numDigits-DisplayEdges:])
		fmt.Fprintln(out, "(Utilisez l'option -v ou --verbose pour afficher la valeur complète)")
	} else {
		fmt.Fprintf(out, "F(%d) = %s\n", n, formatNumberString(resultStr))
	}
}

// formatNumberString insère des séparateurs de milliers dans une chaîne numérique.
func formatNumberString(s string) string {
	if len(s) == 0 {
		return ""
	}
	prefix := ""
	if s[0] == '-' {
		prefix = "-"
		s = s[1:]
	}
	n := len(s)
	if n <= 3 {
		return prefix + s
	}

	var builder strings.Builder
	builder.Grow(len(prefix) + n + (n-1)/3)
	builder.WriteString(prefix)

	firstGroupLen := n % 3
	if firstGroupLen == 0 {
		firstGroupLen = 3
	}
	builder.WriteString(s[:firstGroupLen])

	for i := firstGroupLen; i < n; i += 3 {
		builder.WriteByte(',')
		builder.WriteString(s[i : i+3])
	}
	return builder.String()
}