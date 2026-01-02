package cli

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/testutil"
	"github.com/agbru/fibcalc/internal/ui"
)

// Golden file tests for CLI output
// We store expected output string literals here to verify exact formatting.

func TestDisplayResult_Golden(t *testing.T) {
	ui.InitTheme(false) // Disable colors for deterministic output

	tests := []struct {
		name     string
		result   *big.Int
		n        uint64
		duration time.Duration
		verbose  bool
		details  bool
		concise  bool
		expected string
	}{
		{
			name:     "Simple Result",
			result:   big.NewInt(55),
			n:        10,
			duration: 1 * time.Millisecond,
			verbose:  false,
			details:  false,
			concise:  true,
			expected: "Result binary size: 6 bits.\n\n--- Calculated value ---\nF(10) = 55\n",
		},
		{
			name:     "Detailed Result",
			result:   big.NewInt(55),
			n:        10,
			duration: 0, // 0 duration -> < 1µs
			verbose:  false,
			details:  true,
			concise:  false,
			expected: "Result binary size: 6 bits.\n\n--- Detailed result analysis ---\nCalculation time        : < 1µs\nNumber of digits      : 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			DisplayResult(tt.result, tt.n, tt.duration, tt.verbose, tt.details, tt.concise, &buf)
			got := testutil.StripAnsiCodes(buf.String())

			// Normalize line endings if needed
			if got != tt.expected {
				t.Errorf("Golden mismatch for %s.\nWant:\n%q\nGot:\n%q", tt.name, tt.expected, got)
			}
		})
	}
}

func TestDisplayQuietResult_Golden(t *testing.T) {
	ui.InitTheme(false)
	var buf bytes.Buffer
	DisplayQuietResult(&buf, big.NewInt(12345), 10, time.Second, false)
	expected := "12345\n"
	if buf.String() != expected {
		t.Errorf("Golden mismatch quiet. Want %q, Got %q", expected, buf.String())
	}
}
