// Package cli provides the REPL (Read-Eval-Print Loop) functionality
// for interactive Fibonacci calculations.
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/fibcalc/internal/fibonacci"
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
			if err == io.EOF {
				fmt.Fprintln(r.out, "\nAu revoir!")
				return
			}
			fmt.Fprintf(r.out, "%sErreur de lecture: %v%s\n", ColorRed(), err, ColorReset())
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
	fmt.Fprintf(r.out, "%s║%s     %s🔢 Fibonacci Calculator - Mode Interactif%s            %s║%s\n",
		ColorCyan(), ColorReset(), ColorBold(), ColorReset(), ColorCyan(), ColorReset())
	fmt.Fprintf(r.out, "%s╚══════════════════════════════════════════════════════════╝%s\n\n", ColorCyan(), ColorReset())
}

// printHelp displays available commands.
func (r *REPL) printHelp() {
	fmt.Fprintf(r.out, "%sCommandes disponibles:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  %scalc <n>%s      - Calcule F(n) avec l'algorithme actuel\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %salgo <name>%s   - Change l'algorithme (%s)\n", ColorYellow(), ColorReset(), r.getAlgoList())
	fmt.Fprintf(r.out, "  %scompare <n>%s   - Compare tous les algorithmes pour F(n)\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %slist%s          - Liste les algorithmes disponibles\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %shex%s           - Active/désactive l'affichage hexadécimal\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %sstatus%s        - Affiche la configuration actuelle\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %shelp%s          - Affiche cette aide\n", ColorYellow(), ColorReset())
	fmt.Fprintf(r.out, "  %sexit%s / %squit%s  - Quitte le mode interactif\n", ColorYellow(), ColorReset(), ColorYellow(), ColorReset())
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
		fmt.Fprintf(r.out, "%sAu revoir!%s\n", ColorGreen(), ColorReset())
		return false
	default:
		// Try to interpret as a number for quick calculation
		if n, err := strconv.ParseUint(cmd, 10, 64); err == nil {
			r.calculate(n)
		} else {
			fmt.Fprintf(r.out, "%sCommande inconnue: %s%s\n", ColorRed(), cmd, ColorReset())
			fmt.Fprintf(r.out, "Tapez %shelp%s pour voir les commandes disponibles.\n", ColorYellow(), ColorReset())
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
		fmt.Fprintf(r.out, "%sValeur invalide: %s%s\n", ColorRed(), args[0], ColorReset())
		return
	}

	r.calculate(n)
}

// calculate performs a Fibonacci calculation with the current algorithm.
func (r *REPL) calculate(n uint64) {
	calc, ok := r.registry[r.currentAlgo]
	if !ok {
		fmt.Fprintf(r.out, "%sAlgorithme non trouvé: %s%s\n", ColorRed(), r.currentAlgo, ColorReset())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.config.Timeout)
	defer cancel()

	fmt.Fprintf(r.out, "Calcul de F(%s%d%s) avec %s%s%s...\n",
		ColorMagenta(), n, ColorReset(),
		ColorCyan(), calc.Name(), ColorReset())

	opts := fibonacci.Options{
		ParallelThreshold: r.config.Threshold,
		FFTThreshold:      r.config.FFTThreshold,
	}

	// Create a progress channel (we discard progress in REPL mode for simplicity)
	progressChan := make(chan fibonacci.ProgressUpdate, 10)
	go func() {
		for range progressChan {
			// Discard progress updates in interactive mode
		}
	}()

	start := time.Now()
	result, err := calc.Calculate(ctx, progressChan, 0, n, opts)
	duration := time.Since(start)
	close(progressChan)

	if err != nil {
		fmt.Fprintf(r.out, "%sErreur: %v%s\n", ColorRed(), err, ColorReset())
		return
	}

	// Format duration
	durationStr := FormatExecutionDuration(duration)

	// Display result
	fmt.Fprintf(r.out, "\n%sRésultat:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  Temps: %s%s%s\n", ColorGreen(), durationStr, ColorReset())
	fmt.Fprintf(r.out, "  Bits:  %s%d%s\n", ColorCyan(), result.BitLen(), ColorReset())

	resultStr := result.String()
	numDigits := len(resultStr)
	fmt.Fprintf(r.out, "  Chiffres: %s%d%s\n", ColorCyan(), numDigits, ColorReset())

	if r.config.HexOutput {
		fmt.Fprintf(r.out, "  F(%d) = %s0x%s%s\n", n, ColorGreen(), result.Text(16), ColorReset())
	} else if numDigits > TruncationLimit {
		fmt.Fprintf(r.out, "  F(%d) = %s%s...%s%s (tronqué)\n",
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
		fmt.Fprintf(r.out, "Algorithmes disponibles: %s\n", r.getAlgoList())
		return
	}

	name := strings.ToLower(args[0])
	if _, ok := r.registry[name]; !ok {
		fmt.Fprintf(r.out, "%sAlgorithme inconnu: %s%s\n", ColorRed(), name, ColorReset())
		fmt.Fprintf(r.out, "Algorithmes disponibles: %s\n", r.getAlgoList())
		return
	}

	r.currentAlgo = name
	fmt.Fprintf(r.out, "Algorithme changé en: %s%s%s\n", ColorGreen(), r.registry[name].Name(), ColorReset())
}

// cmdCompare handles the "compare" command.
func (r *REPL) cmdCompare(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.out, "%sUsage: compare <n>%s\n", ColorRed(), ColorReset())
		return
	}

	n, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(r.out, "%sValeur invalide: %s%s\n", ColorRed(), args[0], ColorReset())
		return
	}

	fmt.Fprintf(r.out, "\n%sComparaison pour F(%d):%s\n", ColorBold(), n, ColorReset())
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
			fmt.Fprintf(r.out, "  %s%-20s%s: %sErreur - %v%s\n",
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
			status = ColorRed() + "✗ INCOHÉRENT" + ColorReset()
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
	fmt.Fprintf(r.out, "\n%sAlgorithmes disponibles:%s\n", ColorBold(), ColorReset())
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
	status := "désactivé"
	if r.config.HexOutput {
		status = "activé"
	}
	fmt.Fprintf(r.out, "Affichage hexadécimal: %s%s%s\n", ColorGreen(), status, ColorReset())
}

// cmdStatus displays current REPL configuration.
func (r *REPL) cmdStatus() {
	fmt.Fprintf(r.out, "\n%sConfiguration actuelle:%s\n", ColorBold(), ColorReset())
	fmt.Fprintf(r.out, "  Algorithme:     %s%s%s\n", ColorCyan(), r.currentAlgo, ColorReset())
	fmt.Fprintf(r.out, "  Timeout:        %s%s%s\n", ColorCyan(), r.config.Timeout, ColorReset())
	fmt.Fprintf(r.out, "  Threshold:      %s%d%s bits\n", ColorCyan(), r.config.Threshold, ColorReset())
	fmt.Fprintf(r.out, "  FFT Threshold:  %s%d%s bits\n", ColorCyan(), r.config.FFTThreshold, ColorReset())
	hexStatus := "non"
	if r.config.HexOutput {
		hexStatus = "oui"
	}
	fmt.Fprintf(r.out, "  Hexadécimal:    %s%s%s\n", ColorCyan(), hexStatus, ColorReset())
	fmt.Fprintln(r.out)
}
