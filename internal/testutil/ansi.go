// Package testutil provides shared testing utilities used across the project.
package testutil

import "regexp"

// ansiRegex matches ANSI escape codes for stripping from output.
// This pattern covers:
// - CSI (Control Sequence Introducer) sequences
// - OSC (Operating System Command) sequences
// - Single-character control codes
var ansiRegex = regexp.MustCompile(`[\x1B\x9B][[\\]()#;?]*(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?\x07|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PR-TZcf-ntqry=><~]))`)

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

