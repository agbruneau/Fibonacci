// Package cli provides the REPL (Read-Eval-Print Loop) functionality
// for interactive Fibonacci calculations.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
)

// REPLConfig holds configuration for the REPL session.
type REPLConfig struct {
	// DefaultAlgo is the default algorithm to use for calculations.
	DefaultAlgo string
	// Timeout is the maximum duration for each calculation.
	Timeout time.Duration
	// Threshold is the parallelism threshold.
	Threshold int
	// FFTThreshold is the FFT multiplication threshold.
	FFTThreshold int
	// HexOutput displays results in hexadecimal format.
	HexOutput bool
}

// REPL represents an interactive Fibonacci calculator session.
type REPL struct {
	config      REPLConfig
	registry    map[string]fibonacci.Calculator
	currentAlgo string
	in          io.Reader
	out         io.Writer
}

// NewREPL creates a new REPL instance.
//
// Parameters:
//   - registry: Map of available calculators.
//   - config: REPL configuration.
//
// Returns:
//   - *REPL: A new REPL instance.
func NewREPL(registry map[string]fibonacci.Calculator, config REPLConfig) *REPL {
	currentAlgo := config.DefaultAlgo
	if currentAlgo == "" || currentAlgo == "all" {
		// Pick the first available algorithm as default
		for name := range registry {
			currentAlgo = name
			break
		}
	}

	return &REPL{
		config:      config,
		registry:    registry,
		currentAlgo: currentAlgo,
		in:          os.Stdin,
		out:         os.Stdout,
	}
}

// SetInput sets a custom input reader (useful for testing).
func (r *REPL) SetInput(in io.Reader) {
	r.in = in
}

// SetOutput sets a custom output writer (useful for testing).
func (r *REPL) SetOutput(out io.Writer) {
	r.out = out
}

// Start begins the interactive REPL session.
// It continuously reads user input and processes commands until
// the user exits or EOF is reached.
func (r *REPL) Start() {
	r.printBanner()
	r.printHelp()
	fmt.Fprintln(r.out)

	reader := bufio.NewReader(r.in)

	for {
		fmt.Fprint(r.out, ColorGreen()+"fib> "+ColorReset())

		input, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(r.out, "\nGoodbye!")
				return
			}
			fmt.Fprintf(r.out, "%sRead error: %v%s\n", ColorRed(), err, ColorReset())
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if !r.processCommand(input) {
			return // Exit command received
		}
	}
}

// printBanner displays the REPL welcome banner.
func (r *REPL) printBanner() {
	fmt.Fprintf(r.out, "\n%s╔══════════════════════════════════════════════════════════╗%s\n", ColorCyan(), ColorReset())
	fmt.Fprintf(r.out, "%s║%s     %s🔢 Fibonacci Calculator - Interactive Mode%s            %s║%s\n",
		ColorCyan(), ColorReset(), ColorBold(), ColorReset(), ColorCyan(), ColorReset())
	fmt.Fprintf(r.out, "%s╚══════════════════════════════════════════════════════════╝%s\n\n", ColorCyan(), ColorReset())
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	fmt.Fprintf(r.out, "%sAvailable commands:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  %scalc <n>%s      - Calculate F(n) with current algorithm\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %salgo <name>%s   - Change algorithm (%s)\n", ColorYellow(), ColorReset(), r.getAlgoList())
	fmt.Fprintf(r.out, "  %scompare <n>%s   - Compare all algorithms for F(n)\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %slist%s          - List available algorithms\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %shex%s           - Toggle hexadecimal display\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %sstatus%s        - Display current configuration\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %shelp%s          - Display this help\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %sexit%s / %squit%s  - Exit interactive mode\n", ColorYellow(), ColorReset(), ColorYellow(), ColorReset())
}

// getAlgoList returns a comma-separated list of available algorithms.
func (r *REPL) getAlgoList() string {
	algos := make([]string, 0, len(r.registry))
	for name := range r.registry {
		algos = append(algos, name)
	}
	return strings.Join(algos, ", ")
}

// processCommand parses and executes a user command.
// Returns false if the REPL should exit.
func (r *REPL) processCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "calc", "c":
		r.cmdCalc(args)
	case "algo", "a":
		r.cmdAlgo(args)
	case "compare", "cmp":
		r.cmdCompare(args)
	case "list", "ls":
		r.cmdList()
	case "hex":
		r.cmdHex()
	case "status", "st":
		r.cmdStatus()
	case "help", "h", "?":
		r.printHelp()
	case "exit", "quit", "q":
		fmt.Fprintf(r.out, "%sGoodbye!%s\n", ColorGreen(), ColorReset())
		return false
	default:
		// Try to interpret as a number for quick calculation
		if n, err := strconv.ParseUint(cmd, 10, 64); err == nil {
			r.calculate(n)
		} else {
			fmt.Fprintf(r.out, "%sUnknown command: %s%s\n", ColorRed(), cmd, ColorReset())
			fmt.Fprintf(r.out, "Type %shelp%s to see available commands.\n", ColorYellow(), ColorReset())
		}
	}

	return true
}

// cmdCalc handles the "calc" command.
func (r *REPL) cmdCalc(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.out, "%sUsage: calc <n>%s\n", ColorRed(), ColorReset())
		return
	}

	n, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(r.out, "%sInvalid value: %s%s\n", ColorRed(), args[0], ColorReset())
		return
	}

	r.calculate(n)
}

// calculate performs a Fibonacci calculation with the current algorithm.
func (r *REPL) calculate(n uint64) {
	calc, ok := r.registry[r.currentAlgo]
	if !ok {
		fmt.Fprintf(r.out, "%sAlgorithm not found: %s%s\n", ColorRed(), r.currentAlgo, ColorReset())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	fmt.Fprintf(r.out, "Calculating F(%s%d%s) with %s%s%s...\n",
		ColorMagenta(), n, ColorReset(),
		ColorCyan(), calc.Name(), ColorReset())

	opts := fibonacci.Options{
		ParallelThreshold: r.config.Threshold,
		FFTThreshold:      r.config.FFTThreshold,
	}

	// Create a progress channel
	progressChan := make(chan fibonacci.ProgressUpdate, 10)

	// Use DisplayProgress to show a spinner and progress bar
	var wg sync.WaitGroup
	wg.Add(1)
	go DisplayProgress(&wg, progressChan, 1, r.out)

	start := time.Now()
	result, err := calc.Calculate(ctx, progressChan, 0, n, opts)
	duration := time.Since(start)
	close(progressChan)
	wg.Wait()

	if err != nil {
		fmt.Fprintf(r.out, "%sError: %v%s\n", ColorRed(), err, ColorReset())
		return
	}

	// Format duration
	durationStr := FormatExecutionDuration(duration)

	// Display result
	fmt.Fprintf(r.out, "\n%sResult:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  Time: %s%s%s\n", ColorGreen(), durationStr, ColorReset())
	fmt.Fprintf(r.out, "  Bits:  %s%d%s\n", ColorCyan(), result.BitLen(), ColorReset())

	resultStr := result.String()
	numDigits := len(resultStr)
	fmt.Fprintf(r.out, "  Digits: %s%d%s\n", ColorCyan(), numDigits, ColorReset())

	if r.config.HexOutput {
		fmt.Fprintf(r.out, "  F(%d) = %s0x%s%s\n", n, ColorGreen(), result.Text(16), ColorReset())
	} else if numDigits > TruncationLimit {
		fmt.Fprintf(r.out, "  F(%d) = %s%s...%s%s (truncated)\n",
			n, ColorGreen(), resultStr[:DisplayEdges], resultStr[numDigits-DisplayEdges:], ColorReset())
	} else {
		fmt.Fprintf(r.out, "  F(%d) = %s%s%s\n", n, ColorGreen(), resultStr, ColorReset())
	}
	fmt.Fprintln(r.out)
}

// cmdAlgo handles the "algo" command.
func (r *REPL) cmdAlgo(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.out, "%sUsage: algo <name>%s\n", ColorRed(), ColorReset())
		fmt.Fprintf(r.out, "Available algorithms: %s\n", r.getAlgoList())
		return
	}

	name := strings.ToLower(args[0])
	if _, ok := r.registry[name]; !ok {
		fmt.Fprintf(r.out, "%sUnknown algorithm: %s%s\n", ColorRed(), name, ColorReset())
		fmt.Fprintf(r.out, "Available algorithms: %s\n", r.getAlgoList())
		return
	}

	r.currentAlgo = name
	fmt.Fprintf(r.out, "Algorithm changed to: %s%s%s\n", ColorGreen(), r.registry[name].Name(), ColorReset())
}

// cmdCompare handles the "compare" command.
func (r *REPL) cmdCompare(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.out, "%sUsage: compare <n>%s\n", ColorRed(), ColorReset())
		return
	}

	n, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(r.out, "%sInvalid value: %s%s\n", ColorRed(), args[0], ColorReset())
		return
	}

	fmt.Fprintf(r.out, "\n%sComparison for F(%d):%s\n", ColorBold(), n, ColorReset())
	fmt.Fprintf(r.out, "%s─────────────────────────────────────────────%s\n", ColorCyan(), ColorReset())

	opts := fibonacci.Options{
		ParallelThreshold: r.config.Threshold,
		FFTThreshold:      r.config.FFTThreshold,
	}

	results := make(map[string]string)
	var firstResult string

	for name, calc := range r.registry {
		ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)

		// Create a progress channel for this calculation
		progressChan := make(chan fibonacci.ProgressUpdate, 10)
		go func() {
			for range progressChan {
				// Discard progress updates
			}
		}()

		start := time.Now()
		result, err := calc.Calculate(ctx, progressChan, 0, n, opts)
		duration := time.Since(start)
		close(progressChan)
		cancel()

		if err != nil {
			fmt.Fprintf(r.out, "  %s%-20s%s: %sError - %v%s\n",
				ColorYellow(), name, ColorReset(),
				ColorRed(), err, ColorReset())
			continue
		}

		durationStr := FormatExecutionDuration(duration)
		resultStr := result.String()
		results[name] = resultStr

		if firstResult == "" {
			firstResult = resultStr
		}

		// Check consistency
		status := ColorGreen() + "✓" + ColorReset()
		if resultStr != firstResult {
			status = ColorRed() + "✗ INCONSISTENT" + ColorReset()
		}

		fmt.Fprintf(r.out, "  %s%-20s%s: %s%12s%s %s\n",
			ColorYellow(), name, ColorReset(),
			ColorCyan(), durationStr, ColorReset(),
			status)
	}

	fmt.Fprintf(r.out, "%s─────────────────────────────────────────────%s\n\n", ColorCyan(), ColorReset())
}

// cmdList handles the "list" command.
func (r *REPL) cmdList() {
	fmt.Fprintf(r.out, "\n%sAvailable algorithms:%s\n", ColorBold(), ColorReset())
	for name, calc := range r.registry {
		marker := "  "
		if name == r.currentAlgo {
			marker = ColorGreen() + "► " + ColorReset()
		}
		fmt.Fprintf(r.out, "%s%s%-10s%s - %s\n", marker, ColorYellow(), name, ColorReset(), calc.Name())
	}
	fmt.Fprintln(r.out)
}

// cmdHex toggles hexadecimal output mode.
func (r *REPL) cmdHex() {
	r.config.HexOutput = !r.config.HexOutput
	status := "disabled"
	if r.config.HexOutput {
		status = "enabled"
	}
	fmt.Fprintf(r.out, "Hexadecimal display: %s%s%s\n", ColorGreen(), status, ColorReset())
}

// cmdStatus displays current REPL configuration.
func (r *REPL) cmdStatus() {
	fmt.Fprintf(r.out, "\n%sCurrent configuration:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  Algorithm:      %s%s%s\n", ColorCyan(), r.currentAlgo, ColorReset())
	fmt.Fprintf(r.out, "  Timeout:        %s%s%s\n", ColorCyan(), r.config.Timeout, ColorReset())
	fmt.Fprintf(r.out, "  Threshold:      %s%d%s bits\n", ColorCyan(), r.config.Threshold, ColorReset())
	fmt.Fprintf(r.out, "  FFT Threshold:  %s%d%s bits\n", ColorCyan(), r.config.FFTThreshold, ColorReset())
	hexStatus := "no"
	if r.config.HexOutput {
		hexStatus = "yes"
	}
	fmt.Fprintf(r.out, "  Hexadecimal:    %s%s%s\n", ColorCyan(), hexStatus, ColorReset())
	fmt.Fprintln(r.out)
}
