//
// MODULE ACADÉMIQUE : VALIDATION DE LA COUCHE DE PRÉSENTATION (UI)
//
// OBJECTIF PÉDAGOGIQUE :
// Ce fichier de test démontre les stratégies de validation pour une couche de présentation
// en ligne de commande (CLI) en Go. Il aborde le défi de tester des fonctions dont les
// effets de bord sont des écritures sur des flux de sortie (`io.Writer`) et qui gèrent
// un état d'affichage dynamique et concurrent.
//
// CONCEPTS DE TEST ILLUSTRÉS :
//  1. INJECTION DE DÉPENDANCES POUR LA TESTABILITÉ : Le principe de conception fondamental
//     qui rend ce module testable est l'injection de dépendances. En acceptant un `io.Writer`
//     comme paramètre, les fonctions peuvent être testées en leur fournissant un `bytes.Buffer`
//     en mémoire au lieu de `os.Stdout`. Cela permet de capturer la sortie générée et d'effectuer
//     des assertions précises sur son contenu.
//  2. TESTS PILOTÉS PAR LES DONNÉES (TABLE-DRIVEN TESTS) : Cette technique est employée pour
//     valider les fonctions pures (`formatNumberString`, `progressBar`) sur un large éventail
//     de cas, y compris les cas nominaux et les cas limites, de manière concise et maintenable.
//  3. TESTS DE TYPE "GOLDEN FILE" (APPROCHE SIMPLIFIÉE) : La fonction `TestDisplayResult` compare
//     la sortie textuelle générée à une chaîne de caractères attendue ("golden string"). Cette
//     méthode est une forme de test d'instantané (snapshot testing) qui permet de détecter
//     toute régression non intentionnelle dans le formatage de la sortie.
//  4. VALIDATION D'UN SYSTÈME CONCURRENT (PRODUCTEUR/CONSOMMATEUR) : Le test
//     `TestDisplayAggregateProgress` est le plus complexe. Il simule le comportement du
//     producteur (qui envoie des mises à jour de progression) et vérifie que le consommateur
//     (`DisplayAggregateProgress`) traite correctement ces messages, réagit à la fermeture
//     du canal et se synchronise correctement via un `WaitGroup`.
//
package cli

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// TestFormatNumberString valide la fonction de formatage de nombres par l'ajout de séparateurs de milliers.
func TestFormatNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Chaîne vide", "", ""},
		{"Nombre à un chiffre", "1", "1"},
		{"Nombre à trois chiffres", "123", "123"},
		{"Nombre à quatre chiffres", "1234", "1,234"},
		{"Nombre à six chiffres", "123456", "123,456"},
		{"Nombre à sept chiffres", "1234567", "1,234,567"},
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

// TestProgressBar valide la génération de la représentation textuelle de la barre de progression.
func TestProgressBar(t *testing.T) {
	testCases := []struct {
		name     string
		progress float64
		length   int
		expected string
	}{
		{"Progression nulle (0%)", 0.0, 10, "░░░░░░░░░░"},
		{"Progression partielle (50%)", 0.5, 10, "█████░░░░░"},
		{"Progression complète (100%)", 1.0, 10, "██████████"},
		{"Progression de 25% sur une barre de 20", 0.25, 20, "█████░░░░░░░░░░░░░░░"},
		{"Cas limite : progression > 100%", 1.1, 10, "██████████"},
		{"Cas limite : progression < 0%", -0.1, 10, "░░░░░░░░░░"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := progressBar(tc.progress, tc.length); got != tc.expected {
				t.Errorf("progressBar(%.2f, %d) = %q; attendu %q", tc.progress, tc.length, got, tc.expected)
			}
		})
	}
}

// TestDisplayResult vérifie l'exactitude de la sortie formatée pour le résultat final.
func TestDisplayResult(t *testing.T) {
	duration := 123 * time.Millisecond
	result, _ := new(big.Int).SetString("12586269025", 10) // F(50)

	t.Run("Sortie sans détails", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, false, false, &buf)
		output := buf.String()
		if !strings.Contains(output, "Taille Binaire du Résultat : 34 bits.") {
			t.Errorf("La sortie de base est incorrecte. Attendu: 'Taille Binaire du Résultat : 34 bits.', Obtenu: %q", output)
		}
		if !strings.Contains(output, "(Utilisez l'option -d ou --details") {
			t.Errorf("La sortie de base devrait contenir l'aide pour le mode détails. Obtenu: %q", output)
		}
	})

	t.Run("Sortie détaillée mais non-verbeuse (troncature)", func(t *testing.T) {
		var buf bytes.Buffer
		longNumStr := strings.Repeat("1", 101) // Chaîne plus longue que TruncationLimit
		longResult, _ := new(big.Int).SetString(longNumStr, 10)
		DisplayResult(longResult, 500, duration, false, true, &buf)
		output := buf.String()

		if !strings.Contains(output, "(tronqué)") {
			t.Errorf("La sortie détaillée non-verbeuse devrait être tronquée. Obtenu: %q", output)
		}
		expectedTruncated := fmt.Sprintf("F(500) (tronqué) = %s...%s", longNumStr[:DisplayEdges], longNumStr[len(longNumStr)-DisplayEdges:])
		if !strings.Contains(output, expectedTruncated) {
			t.Errorf("Le format de la sortie tronquée est incorrect.\nAttendu (contenant): %q\nObtenu: %s", expectedTruncated, output)
		}
	})

	t.Run("Sortie détaillée et verbeuse (complète)", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, true, true, &buf)
		output := buf.String()

		if strings.Contains(output, "(tronqué)") {
			t.Errorf("La sortie verbeuse ne devrait pas être tronquée. Obtenu: %q", output)
		}
		// Le formatage ajoute un saut de ligne avant la valeur.
		expectedValue := "F(50) =\n12,586,269,025"
		if !strings.Contains(output, expectedValue) {
			t.Errorf("La valeur dans la sortie verbeuse est incorrecte.\nAttendu (contenant): %q\nObtenu: %s", expectedValue, output)
		}
	})
}

// TestDisplayAggregateProgress valide le comportement du consommateur de l'interface de progression.
func TestDisplayAggregateProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2

	wg.Add(1)
	go DisplayAggregateProgress(&wg, progressChan, numCalculators, &buf)

	// Simulation du producteur : envoi de mises à jour de progression.
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	// NOTE DE TEST : L'attente `time.Sleep` est une simplification. Dans un cadre de
	// test industriel, on utiliserait des horloges simulées ("mock clocks") pour
	// contrôler le temps de manière déterministe et éviter les tests fragiles ("flaky tests").
	time.Sleep(ProgressRefreshRate * 2)

	// Le producteur signale la fin des mises à jour en fermant le canal.
	close(progressChan)
	// Attente de la terminaison de la goroutine du consommateur.
	wg.Wait()

	output := buf.String()

	// La sortie est dynamique et utilise des retours chariot. La validation se concentre
	// sur la dernière ligne affichée, qui représente l'état final.
	// La progression moyenne de 0.25 et 0.50 est 0.375.
	expectedFinalLine := fmt.Sprintf("Progression Moyenne :  37.50%% [%s]", progressBar(0.375, ProgressBarWidth))

	// On ne garde que la dernière ligne de la sortie pour la comparaison,
	// afin d'ignorer les rafraîchissements intermédiaires.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := ""
	if len(lines) > 0 {
		// La dernière ligne peut contenir des codes de contrôle. On les supprime.
		lastLineWithControl := lines[len(lines)-1]
		// On ne garde que ce qui suit le dernier retour chariot pour avoir la ligne finale.
		if finalCR := strings.LastIndex(lastLineWithControl, "\r"); finalCR != -1 {
			lastLine = lastLineWithControl[finalCR+1:]
		} else {
			lastLine = lastLineWithControl
		}
		// On supprime les autres codes ANSI.
		lastLine = strings.TrimPrefix(lastLine, "\033[K")
	}

	if lastLine != expectedFinalLine {
		t.Errorf("La ligne finale de la barre de progression est incorrecte.\nAttendu: %q\nObtenu : %q", expectedFinalLine, lastLine)
	}
}