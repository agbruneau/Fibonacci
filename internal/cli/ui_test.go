//
// MODULE ACADÉMIQUE : TESTS DE LA COUCHE DE PRÉSENTATION (UI)
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier de test illustre comment tester efficacement une couche de présentation
// en ligne de commande (CLI) en Go. Le défi principal est de tester des fonctions
// qui écrivent dans des `io.Writer` (comme le terminal) et qui gèrent des états
// d'affichage complexes.
//
// CONCEPTS CLÉS DÉMONTRÉS :
//  1. INJECTION DE DÉPENDANCES POUR LA TESTABILITÉ : Les fonctions du module `cli`
//     acceptent un `io.Writer` comme argument. En test, au lieu de `os.Stdout`,
//     on injecte un `bytes.Buffer`, ce qui nous permet de capturer la sortie
//     générée et de faire des assertions précises sur son contenu.
//  2. TESTS DE TABLE (TABLE-DRIVEN TESTS) : Utilisés intensivement pour couvrir
//     de nombreux cas de manière concise et lisible, notamment pour `TestFormatNumberString`
//     et `TestProgressBar`.
//  3. TESTS BASÉS SUR DES "GOLDEN FILES" (APPROCHE SIMPLIFIÉE) : Le test
//     `TestDisplayResult` compare la sortie générée à une chaîne de caractères attendue
//     ("golden string"). Pour des sorties très complexes, cette chaîne pourrait être
//     stockée dans un fichier séparé (`.golden` file), une pratique courante pour
//     les tests de snapshot.
//  4. TESTS DE CONCURRENCE POUR L'UI : Le test `TestDisplayAggregateProgress` est
//     le plus complexe. Il simule le comportement du producteur (`main.go`) et
//     vérifie que le consommateur (`DisplayAggregateProgress`) réagit correctement
//     aux messages et à la fermeture du canal, tout en gérant la synchronisation
//     avec des `WaitGroup`.
//
package cli

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestFormatNumberString utilise un test de table pour valider la fonction
// de formatage de nombres avec des séparateurs de milliers.
func TestFormatNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Chaîne vide", "", ""},
		{"Un chiffre", "1", "1"},
		{"Trois chiffres", "123", "123"},
		{"Quatre chiffres", "1234", "1,234"},
		{"Six chiffres", "123456", "123,456"},
		{"Sept chiffres", "1234567", "1,234,567"},
		{"Nombre négatif", "-1234567", "-1,234,567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatNumberString(tc.input); got != tc.expected {
				t.Errorf("formatNumberString(%q) = %q; attendu %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestProgressBar valide la génération de la barre de progression textuelle.
func TestProgressBar(t *testing.T) {
	testCases := []struct {
		name     string
		progress float64
		length   int
		expected string
	}{
		{"0%", 0.0, 10, "░░░░░░░░░░"},
		{"50%", 0.5, 10, "█████░░░░░"},
		{"100%", 1.0, 10, "██████████"},
		{"25%", 0.25, 20, "█████░░░░░░░░░░░░░░░"},
		{"Cas limite > 100%", 1.1, 10, "██████████"},
		{"Cas limite < 0%", -0.1, 10, "░░░░░░░░░░"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// On remplace les caractères par défaut pour éviter les problèmes d'encodage
			// dans certains terminaux de test.
			const (
				filledChar = '█'
				emptyChar  = '░'
			)
			bar := progressBar(tc.progress, tc.length)
			// Remplacement pour une comparaison robuste
			bar = strings.ReplaceAll(bar, string(filledChar), "█")
			bar = strings.ReplaceAll(bar, string(emptyChar), "░")

			if bar != tc.expected {
				t.Errorf("progressBar(%f, %d) = %q; attendu %q", tc.progress, tc.length, bar, tc.expected)
			}
		})
	}
}


// TestDisplayAggregateProgress teste le consommateur de l'UI de progression.
func TestDisplayAggregateProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2

	wg.Add(1)
	// On lance le consommateur dans sa propre goroutine, comme dans l'application réelle.
	go DisplayAggregateProgress(&wg, progressChan, numCalculators, &buf)

	// Simulation du producteur : envoi de quelques mises à jour.
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	// On attend un peu pour laisser au ticker le temps de se déclencher et d'afficher.
	// C'est une simplification. Des tests plus robustes utiliseraient des horloges mockées.
	time.Sleep(ProgressRefreshRate * 2)

	// Le producteur signale la fin en fermant le canal.
	close(progressChan)
	// On attend que la goroutine du consommateur se termine proprement.
	wg.Wait()

	output := buf.String()

	// Vérification de la sortie finale.
	// La sortie attendue est "Progression Moyenne :  37.50% [███░░░░░░░]"
	// suivie d'un saut de ligne. Le 37.50% vient de la moyenne de 0.25 et 0.50.
	// Comme l'affichage est dynamique avec des retours chariot, on ne vérifie que
	// la présence de la ligne finale, qui est la plus importante.
	expectedFinalLine := fmt.Sprintf("Progression Moyenne :  37.50%% [%s]", progressBar(0.375, ProgressBarWidth))

	// Nettoyage de la sortie pour une comparaison plus facile
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := ""
	if len(lines) > 0 {
		// La dernière ligne pertinente peut être précédée par des retours chariots.
		lastLine = strings.TrimSpace(lines[len(lines)-1])
	}

	// Nettoyage de la ligne attendue
	expectedFinalLine = strings.TrimSpace(expectedFinalLine)


	if !strings.Contains(lastLine, expectedFinalLine) {
		t.Errorf("La sortie finale de la barre de progression est incorrecte.\nAttendu (contenant): %q\nObtenu (dernière ligne) : %q", expectedFinalLine, lastLine)
	}
}