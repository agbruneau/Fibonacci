package testutil

import "regexp"

// ansiRegex matches ANSI escape codes for stripping from output.
// This pattern matches the Control Sequence Introducer (CSI) sequences which generally start with ESC [
// and end with a letter, possibly with intermediate characters.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripAnsiCodes removes ANSI escape codes from a string.
// This is useful for testing CLI output without color codes interfering
// with assertions.
//
// Parameters:
//   - s: The string potentially containing ANSI escape codes.
//
// Returns:
//   - string: The input string with all ANSI escape codes removed.
func StripAnsiCodes(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
