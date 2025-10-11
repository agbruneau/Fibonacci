//
// MODULE ACADÉMIQUE : COUCHE DE PRÉSENTATION POUR INTERFACE EN LIGNE DE COMMANDE (CLI)
//
// OBJECTIF PÉDAGOGIQUE :
// Ce module, `cli`, constitue une étude de cas pour la conception d'une interface utilisateur
// en ligne de commande robuste et performante en langage Go. Il met en exergue des techniques
// pour la gestion de l'affichage concurrent, l'interaction non-bloquante, et la présentation
// formatée de données numériques de grande taille.
//
// CONCEPTS ARCHITECTURAUX ET TECHNIQUES ILLUSTRÉS :
//  1. PRINCIPE DE SÉPARATION DES RESPONSABILITÉS (SoC) : La couche de présentation (`cli`) est
//     rigoureusement découplée du domaine métier (`fibonacci`). La communication s'effectue
//     exclusivement via des canaux et des structures de données bien définies (`fibonacci.ProgressUpdate`),
//     ce qui est un prérequis pour la modularité, la testabilité et la maintenabilité du système.
//  2. PATRON DE CONCEPTION PRODUCTEUR/CONSOMMATEUR : Les algorithmes de calcul agissent en tant que
//     "Producteurs" de notifications de progression. La fonction `DisplayAggregateProgress` incarne
//     le "Consommateur", s'exécutant dans une goroutine dédiée pour traiter ces messages de manière
//     asynchrone, découplant ainsi le calcul de l'affichage.
//  3. SÉRIALISATION DE L'ACCÈS À L'ÉTAT : La gestion de l'état partagé (`ProgressState`) est
//     rendue intrinsèquement sûre (thread-safe) sans recours à des primitives de verrouillage
//     explicites (telles que les mutex). L'architecture garantit qu'une unique goroutine
//     (le consommateur) modifie l'état, éliminant par conception les conditions de course (race conditions).
//  4. LIMITATION DE FRÉQUENCE (RATE LIMITING) : L'emploi d'un `time.Ticker` pour réguler la
//     fréquence de rafraîchissement du terminal est une technique essentielle pour prévenir le
//     scintillement de l'affichage ("flickering") et pour minimiser la consommation de ressources CPU.
//  5. CONTRÔLE DU TERMINAL VIA SÉQUENCES D'ÉCHAPPEMENT ANSI : L'utilisation de codes de contrôle
//     (`\r`, `\033[K`) illustre la création d'interfaces dynamiques et professionnelles au sein
//     d'un environnement textuel standard.
//
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

// Constantes définissant les paramètres de l'interface utilisateur.
const (
	// ProgressRefreshRate détermine la cadence de rafraîchissement de la barre de progression.
	// Une fréquence de 10 Hz (100ms) représente un compromis optimal entre la fluidité
	// de l'animation perçue par l'utilisateur et la charge de traitement induite.
	ProgressRefreshRate = 100 * time.Millisecond

	// ProgressBarWidth définit la largeur, en nombre de caractères, de la barre de progression.
	ProgressBarWidth = 40

	// TruncationLimit seuil (en nombre de chiffres) au-delà duquel la représentation
	// décimale d'un résultat est tronquée pour préserver la lisibilité de l'affichage.
	TruncationLimit = 100

	// DisplayEdges spécifie le nombre de chiffres à conserver au début et à la fin
	// d'un nombre dont l'affichage est tronqué.
	DisplayEdges = 25
)

// ProgressState encapsule l'état agrégé de la progression de l'ensemble des calculs.
// Cette structure sert de "modèle" dans une architecture MVC simplifiée, détenant les données
// qui seront visualisées.
type ProgressState struct {
	progresses     []float64 // Stocke la valeur de progression [0.0, 1.0] pour chaque calculateur.
	numCalculators int
	out            io.Writer // Destination de la sortie (e.g., os.Stdout), injectée pour faciliter les tests.
}

// NewProgressState est une fonction de fabrique qui initialise un nouvel état de progression.
func NewProgressState(numCalculators int, out io.Writer) *ProgressState {
	return &ProgressState{
		progresses:     make([]float64, numCalculators),
		numCalculators: numCalculators,
		out:            out,
	}
}

// Update met à jour la valeur de progression pour un calculateur identifié par son index.
// Cette méthode est exclusivement appelée par la goroutine du consommateur, ce qui garantit
// un accès séquentiel et prévient les conditions de course sans nécessiter de verrou.
func (ps *ProgressState) Update(index int, value float64) {
	if index >= 0 && index < len(ps.progresses) {
		ps.progresses[index] = value
	}
}

// CalculateAverage calcule la moyenne arithmétique des progressions individuelles.
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

// PrintBar génère et affiche la représentation textuelle de la barre de progression.
func (ps *ProgressState) PrintBar(final bool) {
	avgProgress := ps.CalculateAverage()
	label := "Progression"
	if ps.numCalculators > 1 {
		label = "Progression Moyenne"
	}
	bar := progressBar(avgProgress, ProgressBarWidth)

	// NOTE TECHNIQUE : La séquence `\r\033[K` est une méthode robuste pour rafraîchir une ligne.
	// `\r` (Retour Chariot) déplace le curseur au début de la ligne.
	// `\033[K` (Erase in Line) efface le contenu de la ligne à partir du curseur.
	// Cette combinaison garantit que l'ancienne ligne est entièrement effacée avant l'écriture de la nouvelle.
	fmt.Fprintf(ps.out, "\r\033[K%s : %6.2f%% [%s]", label, avgProgress*100, bar)

	if final {
		fmt.Fprintln(ps.out) // Ajoute un saut de ligne pour conserver l'affichage final.
	}
}

// DisplayAggregateProgress est la fonction principale du consommateur de l'UI.
// Exécutée dans une goroutine, elle orchestre le cycle de vie de l'affichage de la progression.
func DisplayAggregateProgress(wg *sync.WaitGroup, progressChan <-chan fibonacci.ProgressUpdate, numCalculators int, out io.Writer) {
	defer wg.Done() // Garantit la notification de terminaison au WaitGroup appelant.

	// Cas particulier : si aucun calculateur n'est actif, il est impératif de "drainer"
	// le canal pour permettre au producteur de se terminer sans blocage.
	if numCalculators <= 0 {
		for range progressChan { // Lit et ignore tous les messages jusqu'à la fermeture du canal.
		}
		return
	}

	state := NewProgressState(numCalculators, out)
	ticker := time.NewTicker(ProgressRefreshRate)
	defer ticker.Stop() // Libère les ressources associées au ticker.

	// Boucle d'événements principale, gérant les sources asynchrones.
	for {
		select {
		// Cas 1 : Réception d'une mise à jour de progression.
		case update, ok := <-progressChan:
			if !ok {
				// `ok` est faux si le canal est fermé, signalant la fin des calculs.
				state.PrintBar(true) // Affichage final.
				return               // Termine la goroutine.
			}
			state.Update(update.CalculatorIndex, update.Value)

		// Cas 2 : Un événement de rafraîchissement a été émis par le ticker.
		case <-ticker.C:
			state.PrintBar(false)
		}
	}
}

// progressBar génère la chaîne de caractères représentant une barre de progression.
func progressBar(progress float64, length int) string {
	if progress > 1.0 { progress = 1.0 }
	if progress < 0.0 { progress = 0.0 }

	const (
		filledChar = '█' // U+2588 : Bloc plein
		emptyChar  = '░' // U+2591 : Ombre légère
	)
	count := int(progress * float64(length))

	// `strings.Builder` est utilisé pour une construction de chaîne performante,
	// minimisant les allocations mémoire par rapport à la concaténation standard.
	var builder strings.Builder
	// Pré-allocation du buffer pour optimiser davantage en évitant les réallocations dynamiques.
	builder.Grow(length * 3) // UTF-8 runes can take up to 3 bytes.

	for i := 0; i < length; i++ {
		if i < count {
			builder.WriteRune(filledChar)
		} else {
			builder.WriteRune(emptyChar)
		}
	}
	return builder.String()
}

// DisplayResult formate et affiche le résultat final du calcul ainsi que des métadonnées associées.
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

	// NOTE DE PERFORMANCE : La conversion d'un `big.Int` en chaîne décimale est une
	// opération coûteuse (complexité super-linéaire). Elle ne doit être effectuée qu'une seule fois.
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

// formatNumberString insère des séparateurs de milliers dans une chaîne numérique pour en améliorer la lisibilité.
func formatNumberString(s string) string {
	if len(s) == 0 { return "" }
	prefix := ""
	if s[0] == '-' {
		prefix = "-"
		s = s[1:]
	}
	n := len(s)
	if n <= 3 { return prefix + s }

	var builder strings.Builder
	// Calcul de la capacité exacte du buffer : n chiffres + (n-1)/3 séparateurs.
	builder.Grow(len(prefix) + n + (n-1)/3)
	builder.WriteString(prefix)

	firstGroupLen := n % 3
	if firstGroupLen == 0 { firstGroupLen = 3 }
	builder.WriteString(s[:firstGroupLen])

	for i := firstGroupLen; i < n; i += 3 {
		builder.WriteByte(',')
		builder.WriteString(s[i : i+3])
	}
	return builder.String()
}