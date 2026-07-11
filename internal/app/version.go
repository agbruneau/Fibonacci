package app

import (
	"fmt"
	"io"
	"runtime"
)

// Build-time variables set via -ldflags.
// These are populated during builds to provide version information.
//
// Example build command:
//
//	go build -ldflags="-X github.com/agbruneau/FibGo/internal/app.Version=v1.2.3 -X github.com/agbruneau/FibGo/internal/app.Commit=abc123 -X github.com/agbruneau/FibGo/internal/app.BuildDate=2025-01-01T00:00:00Z"
var (
	// Version is the semantic version of the application (e.g., "v1.0.0").
	Version = "dev"
	// Commit is the short Git commit hash (e.g., "abc123").
	Commit = "unknown"
	// BuildDate is the ISO 8601 timestamp of the build (e.g., "2025-01-01T00:00:00Z").
	BuildDate = "unknown"
)

// HasVersionFlag checks if any argument is a version flag.
// This allows --version to work in any position (e.g., "fibcalc -n 100000 --version").
//
// Parameters:
//   - args: The command-line arguments to check (typically os.Args[1:]).
//
// Returns:
//   - bool: True if a version flag is found, false otherwise.
func HasVersionFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--version" || arg == "-version" || arg == "-V" {
			return true
		}
	}
	return false
}

// PrintVersion outputs version information to the given writer.
// It displays the application version, commit hash, build date,
// Go version, and OS/architecture.
//
// Parameters:
//   - out: The writer to output version information to.
func PrintVersion(out io.Writer) {
	fmt.Fprintf(out, "fibcalc %s\n", Version)
	fmt.Fprintf(out, "  Commit:     %s\n", Commit)
	fmt.Fprintf(out, "  Built:      %s\n", BuildDate)
	fmt.Fprintf(out, "  Go version: %s\n", runtime.Version())
	fmt.Fprintf(out, "  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
