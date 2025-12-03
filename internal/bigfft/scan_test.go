package bigfft

import (
	"math/big"
	"testing"
)


func TestFromDecimalString(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"SingleDigit", "1"},
		{"Small", "12345"},
		{"Threshold", string(make([]byte, quadraticScanThreshold))}, // Need logic to fill with digits
		{"Large", "123456789012345678901234567890"},
	}

	// Create threshold string
	for i := range tests {
		if tests[i].name == "Threshold" {
			// quadraticScanThreshold is 1232
			tests[i].in = string(make([]byte, quadraticScanThreshold))
			for j := 0; j < quadraticScanThreshold; j++ {
				// We can't assign to string index directly, but we can build it.
				// Actually using strings.Repeat is easiest if we want repeated char.
				// But original loop appended "9".
			}
			// Optimized construction
			b := make([]byte, quadraticScanThreshold)
			for j := range b {
				b[j] = '9'
			}
			tests[i].in = string(b)
		}
	}

	// Add a recursive case (larger than threshold)
	// Optimized construction
	count := quadraticScanThreshold * 3
	b := make([]byte, count+1)
	b[0] = '1'
	for i := 1; i <= count; i++ {
		b[i] = '0'
	}
	tests = append(tests, struct{name, in string}{"Recursive", string(b)})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromDecimalString(tt.in)

			// Verify against standard library
			expected := new(big.Int)
			expected.SetString(tt.in, 10)

			if got.Cmp(expected) != 0 {
				t.Errorf("FromDecimalString(%q) = %v, want %v", tt.in, got, expected)
			}
		})
	}
}
