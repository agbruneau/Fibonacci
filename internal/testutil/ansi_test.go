package testutil

import "testing"

func TestStripAnsiCodes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No codes",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "Simple color",
			input:    "\x1b[31mRed\x1b[0m",
			expected: "Red",
		},
		{
			name:     "Bold and color",
			input:    "\x1b[1;32mGreen Bold\x1b[0m",
			expected: "Green Bold",
		},
		{
			name:     "Multiple codes",
			input:    "Normal \x1b[33mYellow\x1b[0m \x1b[34mBlue\x1b[0m",
			expected: "Normal Yellow Blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StripAnsiCodes(tt.input)
			if got != tt.expected {
				t.Errorf("StripAnsiCodes(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
