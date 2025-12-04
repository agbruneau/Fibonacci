package bigfft

import (
	"math/big"
	"strings"
	"testing"
)

func TestFromDecimalString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Zero", "0", "0"},
		{"One", "1", "1"},
		{"Small number", "123", "123"},
		{"Large number", "123456789012345678901234567890", "123456789012345678901234567890"},
		{"Very large number", strings.Repeat("9", 2000), strings.Repeat("9", 2000)},
		{"Number with leading zeros", "000123", "123"},
		{"Large power of 10", "1" + strings.Repeat("0", 100), "1" + strings.Repeat("0", 100)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FromDecimalString(tc.input)
			expected := new(big.Int)
			expected.SetString(tc.expected, 10)

			if result.Cmp(expected) != 0 {
				t.Errorf("FromDecimalString(%q) = %s, want %s", tc.input, result.String(), expected.String())
			}
		})
	}
}

func TestFromDecimalString_EdgeCases(t *testing.T) {
	t.Run("Empty string", func(t *testing.T) {
		result := FromDecimalString("")
		if result.Sign() != 0 {
			t.Errorf("Empty string should result in zero, got %s", result.String())
		}
	})

	t.Run("Single digit", func(t *testing.T) {
		for i := 0; i <= 9; i++ {
			input := string(rune('0' + i))
			result := FromDecimalString(input)
			expected := big.NewInt(int64(i))
			if result.Cmp(expected) != 0 {
				t.Errorf("FromDecimalString(%q) = %s, want %s", input, result.String(), expected.String())
			}
		}
	})

	t.Run("Very long string", func(t *testing.T) {
		// Test with a string longer than quadraticScanThreshold
		longStr := strings.Repeat("9", 5000)
		result := FromDecimalString(longStr)
		expected := new(big.Int)
		expected.SetString(longStr, 10)
		if result.Cmp(expected) != 0 {
			t.Errorf("Very long string conversion failed")
		}
	})

	t.Run("String just above threshold", func(t *testing.T) {
		// quadraticScanThreshold is 1232, so test with 1233 digits
		longStr := "1" + strings.Repeat("0", 1232)
		result := FromDecimalString(longStr)
		expected := new(big.Int)
		expected.SetString(longStr, 10)
		if result.Cmp(expected) != 0 {
			t.Errorf("String just above threshold conversion failed")
		}
	})

	t.Run("String at threshold", func(t *testing.T) {
		// Test with exactly quadraticScanThreshold digits
		longStr := strings.Repeat("9", quadraticScanThreshold)
		result := FromDecimalString(longStr)
		expected := new(big.Int)
		expected.SetString(longStr, 10)
		if result.Cmp(expected) != 0 {
			t.Errorf("String at threshold conversion failed")
		}
	})
}

func TestFromDecimalString_Consistency(t *testing.T) {
	// Test that FromDecimalString produces the same result as big.Int.SetString
	testStrings := []string{
		"0",
		"1",
		"10",
		"100",
		"1000",
		"123456789",
		strings.Repeat("9", 100),
		strings.Repeat("9", 1000),
		strings.Repeat("9", 2000),
		strings.Repeat("1", 3000),
	}

	for _, s := range testStrings {
		t.Run(s[:min(20, len(s))], func(t *testing.T) {
			result1 := FromDecimalString(s)
			result2 := new(big.Int)
			result2.SetString(s, 10)

			if result1.Cmp(result2) != 0 {
				t.Errorf("FromDecimalString(%q) = %s, but SetString gives %s",
					s, result1.String(), result2.String())
			}
		})
	}
}

func TestFromDecimalString_Performance(t *testing.T) {
	// Test that the function can handle very large inputs
	largeInput := strings.Repeat("9", 10000)
	result := FromDecimalString(largeInput)
	if result.Sign() <= 0 {
		t.Error("Large input should produce a positive number")
	}
	// Verify it's correct by checking it's close to 10^10000 - 1
	expected := new(big.Int)
	expected.SetString(largeInput, 10)
	if result.Cmp(expected) != 0 {
		t.Error("Large input conversion failed")
	}
}
