package cli

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/fibcalc/internal/fibonacci"
)

// MockSpinner is a mock implementation of the Spinner interface for testing.
type MockSpinner struct {
	startCalled bool
	stopCalled  bool
	suffix      string
	mu          sync.Mutex
}

func (ms *MockSpinner) Start() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.startCalled = true
}

func (ms *MockSpinner) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.stopCalled = true
}

func (ms *MockSpinner) UpdateSuffix(suffix string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.suffix = suffix
}

// TestFormatNumberString validates the number formatting function.
func TestFormatNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single-digit number", "1", "1"},
		{"Three-digit number", "123", "123"},
		{"Four-digit number", "1234", "1,234"},
		{"Six-digit number", "123456", "123,456"},
		{"Seven-digit number", "1234567", "1,234,567"},
		{"Negative number", "-1234567", "-1,234,567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatNumberString(tc.input); got != tc.expected {
				t.Errorf("formatNumberString(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

var accentReplacer = strings.NewReplacer(
	"\u00e9", "e", "\u00e8", "e", "\u00ea", "e", "\u00eb", "e",
	"\u00e0", "a", "\u00e2", "a", "\u00e4", "a",
	"\u00f9", "u", "\u00fb", "u", "\u00fc", "u",
	"\u00f4", "o", "\u00f6", "o",
	"\u00ee", "i", "\u00ef", "i",
	"\u00e7", "c",
	"\u00c9", "E", "\u00c8", "E", "\u00ca", "E",
	"\u00c0", "A", "\u00c2", "A",
	"\u00d9", "U", "\u00db", "U",
	"\u00d4", "O",
	"\u0152", "OE", "\u0153", "oe",
	"\u2019", "'", "\u201c", "\"", "\u201d", "\"",
	"\u0300", "", "\u0301", "", "\u0302", "", "\u0308", "",
)

func sanitizeOutput(s string) string {
	const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-ntqry=><~]))"
	re := regexp.MustCompile(ansi)
	cleaned := re.ReplaceAllString(s, "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	return accentReplacer.Replace(cleaned)
}

// TestDisplayResult checks the formatting of the result output.
func TestDisplayResult(t *testing.T) {
	duration := 123 * time.Millisecond
	result, _ := new(big.Int).SetString("12586269025", 10) // F(50)

	t.Run("Output without details", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, false, false, &buf)
		output := sanitizeOutput(buf.String())
		if !strings.Contains(output, "Taille binaire du resultat : 34 bits.") {
			t.Errorf("La sortie de base est incorrecte. Attendu : 'Taille binaire du resultat : 34 bits.'. Obtenu : %q", output)
		}
		if !strings.Contains(output, "(Astuce : utiliser l'option -d") {
			t.Errorf("La sortie de base devrait contenir l'aide pour le mode details. Obtenu : %q", output)
		}
	})

	t.Run("Detailed but non-verbose output (truncation)", func(t *testing.T) {
		var buf bytes.Buffer
		longNumStr := strings.Repeat("1", 101) // String longer than TruncationLimit
		longResult, _ := new(big.Int).SetString(longNumStr, 10)
		DisplayResult(longResult, 500, duration, false, true, &buf)
		output := sanitizeOutput(buf.String())

		if !strings.Contains(output, "(tronque)") {
			t.Errorf("La sortie detaillee non verbeuse devrait etre tronquee. Obtenu : %q", output)
		}
		expectedTruncated := fmt.Sprintf("F(500) (tronque) = %s...%s", longNumStr[:DisplayEdges], longNumStr[len(longNumStr)-DisplayEdges:])
		if !strings.Contains(output, expectedTruncated) {
			t.Errorf("Le format de sortie tronque est incorrect.\nAttendu (contenant) : %q\nObtenu : %s", expectedTruncated, output)
		}
	})

	t.Run("Detailed and verbose output (full)", func(t *testing.T) {
		var buf bytes.Buffer
		DisplayResult(result, 50, duration, true, true, &buf)
		output := sanitizeOutput(buf.String())

		if strings.Contains(output, "(tronque)") {
			t.Errorf("La sortie verbeuse ne doit pas etre tronquee. Obtenu : %q", output)
		}
		expectedValue := "F(50) =\n12,586,269,025"
		if !strings.Contains(output, expectedValue) {
			t.Errorf("La valeur dans la sortie verbeuse est incorrecte.\nAttendu (contenant) : %q\nObtenu : %s", expectedValue, output)
		}
	})
}

// TestDisplayProgress validates the behavior of the progress display.
func TestDisplayProgress(t *testing.T) {
	var buf bytes.Buffer
	var wg sync.WaitGroup
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	numCalculators := 2
	mock := &MockSpinner{}

	originalNewSpinner := newSpinner
	newSpinner = func(out io.Writer) Spinner {
		return mock
	}
	defer func() { newSpinner = originalNewSpinner }()

	wg.Add(1)
	go DisplayProgress(&wg, progressChan, numCalculators, &buf)

	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 0, Value: 0.25}
	progressChan <- fibonacci.ProgressUpdate{CalculatorIndex: 1, Value: 0.50}

	// Give the ticker time to update the suffix.
	time.Sleep(ProgressRefreshRate * 2)

	mock.mu.Lock()
	suffix := accentReplacer.Replace(mock.suffix)
	if !strings.Contains(suffix, "Progression moyenne") {
		t.Errorf("Le suffixe du spinner devrait afficher 'Progression moyenne'. Obtenu : %q", suffix)
	}
	if !strings.Contains(suffix, "37.50%") {
		t.Errorf("Le suffixe du spinner devrait afficher le pourcentage moyen correct. Obtenu : %q", suffix)
	}
	mock.mu.Unlock()

	close(progressChan)
	wg.Wait()

	if !mock.startCalled {
		t.Error("Spinner.Start() was not called.")
	}
	if !mock.stopCalled {
		t.Error("Spinner.Stop() was not called.")
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	finalSuffix := accentReplacer.Replace(mock.suffix)
	if !strings.Contains(finalSuffix, "Calcul termine.") {
		t.Errorf("Le suffixe final du spinner est incorrect. Obtenu : %q", finalSuffix)
	}
}
