package cli

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// TruncationLimit is the digit threshold from which a result is truncated
	// in standard output to avoid cluttering the terminal.
	TruncationLimit = 100
	// DisplayEdges specifies the number of digits to display at the beginning
	// and end of a truncated number.
	DisplayEdges = 25
	// ProgressRefreshRate defines the refresh frequency of the progress bar.
	// Optimized to 200ms to reduce updates and improve performance.
	ProgressRefreshRate = 200 * time.Millisecond
	// ProgressBarWidth defines the width in characters of the progress bar.
	ProgressBarWidth = 40
)

// spinnerFrames is the animation cycle: the Braille set of
// github.com/briandowns/spinner's CharSets[11].
var spinnerFrames = [...]string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressLine draws a single-line, carriage-return-updated status line.
//
// It replaces github.com/briandowns/spinner (audit DEP-02 / CON-04, ADR-0012
// D5). That dependency rendered one line and cost, in exchange:
//
//   - a data race. Its render goroutine read Suffix under its own mutex while
//     running, so writing the field from the caller raced with it (CONC-01).
//     The workaround was to Stop() and Start() the spinner around every suffix
//     write — tearing down and respawning a goroutine five times a second so
//     that a string assignment would be safely ordered.
//   - a Spinner interface, a realSpinner adapter with three function fields,
//     a package-level newSpinner seam, and a test asserting the stop/write/start
//     ordering of the workaround.
//   - four modules in the graph (spinner, fatih/color, mattn/go-colorable,
//     mattn/go-isatty).
//
// There is no goroutine here at all. DisplayProgress already owns a ticker and
// a loop; drawing on that loop is the whole implementation, and the race cannot
// exist because only one goroutine ever touches the writer.
type progressLine struct {
	out   io.Writer
	frame int
	width int // width of the last line drawn, for erasing
}

func newProgressLine(out io.Writer) *progressLine {
	return &progressLine{out: out}
}

// Draw renders the next animation frame followed by text, in place.
func (p *progressLine) Draw(text string) {
	frame := spinnerFrames[p.frame%len(spinnerFrames)]
	p.frame++

	line := frame + text
	// Pad to the previous width so a shrinking line does not leave a tail
	// behind — the ETA field in particular goes from "ETA: 1m20s" to "ETA: < 1s".
	if pad := p.width - utf8.RuneCountInString(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	p.width = utf8.RuneCountInString(line)

	fmt.Fprintf(p.out, "\r%s", line)
}

// Clear erases the line and returns the cursor to its start, so the caller can
// print a final result where the animation was.
func (p *progressLine) Clear() {
	if p.width == 0 {
		return
	}
	fmt.Fprintf(p.out, "\r%s\r", strings.Repeat(" ", p.width))
	p.width = 0
}
