// This file provides a color provider implementation for use with the errors package.

package cli

import (
	apperrors "github.com/agbruneau/FibGo/internal/errors"
	"github.com/agbruneau/FibGo/internal/ui"
)

// Ensure CLIColorProvider implements apperrors.ColorProvider at compile time.
var _ apperrors.ColorProvider = CLIColorProvider{}

// CLIColorProvider implements apperrors.ColorProvider using CLI theme functions.
// It provides terminal color codes for formatted error messages based on the
// current CLI theme settings.
//
// This type is exported to allow usage from other packages (orchestration,
// calibration) without duplicating the implementation.
type CLIColorProvider struct{}

// Yellow returns the yellow color code from the current CLI theme.
//
// Returns:
//   - string: The ANSI escape code for yellow color.
func (c CLIColorProvider) Yellow() string { return ui.ColorYellow() }

// Reset returns the reset color code from the current CLI theme.
//
// Returns:
//   - string: The ANSI escape code to reset colors.
func (c CLIColorProvider) Reset() string { return ui.ColorReset() }
