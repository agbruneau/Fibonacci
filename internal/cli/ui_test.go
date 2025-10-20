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

// TestFormatNumberString valide la fonction de formatage de nombres.
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

// TestProgressBar valide la génération de la barre de progression.
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

// TestDisplayResult vérifie le formatage de la sortie du résultat.
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
		expectedValue := "F(50) =\n12,586,269,025"
		if !strings.Contains(output, expectedValue) {
			t.Errorf("La valeur dans la sortie verbeuse est incorrecte.\nAttendu (contenant): %q\nObtenu: %s", expectedValue, output)
		}
	})
}

// TestDisplayAggregateProgress valide le comportement du consommateur de progression.
func TestDisplayAggregateProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2

	wg.Add(1)
	go DisplayAggregateProgress(&wg, progressChan, numCalculators, &buf)

	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	time.Sleep(ProgressRefreshRate * 2)

	close(progressChan)
	wg.Wait()

	output := buf.String()
	expectedFinalLine := fmt.Sprintf("Progression Moyenne :  37.50%% [%s]", progressBar(0.375, ProgressBarWidth))

	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := ""
	if len(lines) > 0 {
		lastLineWithControl := lines[len(lines)-1]
		if finalCR := strings.LastIndex(lastLineWithControl, "\r"); finalCR != -1 {
			lastLine = lastLineWithControl[finalCR+1:]
		} else {
			lastLine = lastLineWithControl
		}
		lastLine = strings.TrimPrefix(lastLine, "\033[K")
	}

	if lastLine != expectedFinalLine {
		t.Errorf("La ligne finale de la barre de progression est incorrecte.\nAttendu: %q\nObtenu : %q", expectedFinalLine, lastLine)
	}
}