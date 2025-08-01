package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"sort"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Types and Structures
// ------------------------------------------------------------

type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

type task struct {
	name string
	fn   fibFunc
}

type result struct {
	name     string
	value    *big.Int
	duration time.Duration
	err      error
}

// Global configuration flags
var verboseFlag bool

// ------------------------------------------------------------
// Constants for Binet's Formula
// ------------------------------------------------------------

var (
	phi   = big.NewFloat(1.61803398874989484820458683436563811772030917980576)
	sqrt5 = big.NewFloat(2.23606797749978969640917366873127623544061835961152)
	// Log2(Phi) used for precision estimation
	log2Phi = 0.6942419136306173
)

// ------------------------------------------------------------
// Progress Display Management
// ------------------------------------------------------------

type progressData struct {
	name string
	pct  float64
}

// progressPrinter manages the consolidated display of progress.
// IMPROVEMENT: Context-aware to handle timeouts gracefully.
func progressPrinter(ctx context.Context, progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for _, k := range taskNames {
		status[k] = 0.0
	}

	if len(taskNames) == 0 {
		return
	}

	needsUpdate := true

	for {
		select {
		case <-ctx.Done():
			// Context cancelled (e.g., timeout). Print final state and exit.
			printStatus(status, taskNames)
			fmt.Println("\n(Cancelled/Timeout)")
			return
		case p, ok := <-progress:
			if !ok {
				// Channel closed (all tasks finished).
				printStatus(status, taskNames)
				fmt.Println() // Final newline
				return
			}
			if _, exists := status[p.name]; exists && status[p.name] != p.pct {
				status[p.name] = p.pct
				needsUpdate = true
			}
		case <-ticker.C:
			if needsUpdate {
				printStatus(status, taskNames)
				needsUpdate = false
			}
		}
	}
}

// printStatus displays the current progress on a single line.
func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r")
	var parts []string
	for _, k := range keys {
		v, ok := status[k]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%-18s %6.2f%%", k, v))
	}
	output := strings.Join(parts, " | ")
	// Use Printf with padding to clear the line robustly.
	fmt.Printf("%-120s", output)
}

// ------------------------------------------------------------
// Memory Pool Management (sync.Pool for big.Int)
// ------------------------------------------------------------

func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// fastDoublingTemps manages temporary *big.Int used in the Fast Doubling algorithm.
type fastDoublingTemps struct {
	a_orig, t1, t2, t3, new_a, new_b, t_sum *big.Int
}

func (tmp *fastDoublingTemps) acquire(pool *sync.Pool) {
	tmp.a_orig = pool.Get().(*big.Int)
	tmp.t1 = pool.Get().(*big.Int)
	tmp.t2 = pool.Get().(*big.Int)
	tmp.t3 = pool.Get().(*big.Int)
	tmp.new_a = pool.Get().(*big.Int)
	tmp.new_b = pool.Get().(*big.Int)
	tmp.t_sum = pool.Get().(*big.Int)
}

func (tmp *fastDoublingTemps) release(pool *sync.Pool) {
	pool.Put(tmp.a_orig)
	pool.Put(tmp.t1)
	pool.Put(tmp.t2)
	pool.Put(tmp.t3)
	pool.Put(tmp.new_a)
	pool.Put(tmp.new_b)
	pool.Put(tmp.t_sum)
}

// ------------------------------------------------------------
// Helper Functions
// ------------------------------------------------------------

// handleBaseCases checks for n < 0, n=0, and n=1.
func handleBaseCases(n int, progress chan<- float64) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative index not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}
	return nil, nil
}

// ------------------------------------------------------------
// Algorithm 1: Fast Doubling (O(log N)) - OPTIMIZED
// ------------------------------------------------------------

func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// a = F(k), b = F(k+1)
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))
	temps := fastDoublingTemps{}

	// MAJOR OPTIMIZATION: Acquire temporary variables ONCE outside the loop.
	// This significantly reduces sync.Pool contention and overhead compared to the original code.
	temps.acquire(pool)
	defer temps.release(pool)

	for i := totalBits - 1; i >= 0; i-- {
		// Check for cancellation periodically.
		if i%5 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		// F(2k) = F(k) * [2*F(k+1) – F(k)]
		// F(2k+1) = F(k)^2 + F(k+1)^2

		temps.a_orig.Set(a) // Store F(k)

		// Calculate F(2k)
		temps.t1.Lsh(b, 1)                      // t1 = 2*b
		temps.t1.Sub(temps.t1, temps.a_orig)    // t1 = 2*b - a_orig
		temps.new_a.Mul(temps.a_orig, temps.t1) // new_a = F(2k)

		// Calculate F(2k+1)
		temps.t2.Mul(temps.a_orig, temps.a_orig) // t2 = a_orig^2
		temps.t3.Mul(b, b)                       // t3 = b^2
		temps.new_b.Add(temps.t2, temps.t3)      // new_b = F(2k+1)

		a.Set(temps.new_a)
		b.Set(temps.new_b)

		// If the i-th bit of n is 1, advance one step.
		if (uint(n)>>i)&1 == 1 {
			temps.t_sum.Add(a, b) // t_sum = F(2k+2)
			a.Set(b)              // a = F(2k+1)
			b.Set(temps.t_sum)    // b = F(2k+2)
		}

		if progress != nil {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}

	// Return a copy as 'a' belongs to the pool.
	return new(big.Int).Set(a), nil
}

// ------------------------------------------------------------
// Algorithm 2: Matrix Exponentiation (O(log N))
// ------------------------------------------------------------

// [Matrix implementation remains largely the same as it was already well optimized]

type mat2 struct {
	a, b, c, d *big.Int
}

func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int), b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int), d: pool.Get().(*big.Int),
	}
}

func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

func (m *mat2) setIdentity() {
	m.a.SetInt64(1)
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

func (m *mat2) setFibBase() {
	m.a.SetInt64(1)
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// matMul calculates target = m1 * m2. target must be distinct from m1 and m2.
func matMul(target, m1, m2 *mat2, pool *sync.Pool) {
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	// target.a = (m1.a*m2.a) + (m1.b*m2.c)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	target.a.Add(t1, t2)
	// target.b = (m1.a*m2.b) + (m1.b*m2.d)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	target.b.Add(t1, t2)
	// target.c = (m1.c*m2.a) + (m1.d*m2.c)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	target.c.Add(t1, t2)
	// target.d = (m1.c*m2.b) + (m1.d*m2.d)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	target.d.Add(t1, t2)
}

func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// We calculate M^(n-1). F(n) will be in res.a.
	exp := uint(n - 1)

	res := newMat2(pool)
	defer res.release(pool)
	res.setIdentity()

	base := newMat2(pool)
	defer base.release(pool)
	base.setFibBase()

	temp := newMat2(pool)
	defer temp.release(pool)

	totalSteps := bits.Len(exp)
	stepsDone := 0

	// Binary exponentiation
	for exp > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exp&1 == 1 {
			matMul(temp, res, base, pool)
			res.set(temp)
		}

		// Optimization: avoid the final unnecessary squaring.
		if exp > 1 {
			matMul(temp, base, base, pool)
			base.set(temp)
		}

		exp >>= 1
		stepsDone++

		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}

	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Algorithm 3: Binet's Formula (O(log N), Float-based)
// ------------------------------------------------------------

func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	// Determine required precision: n * log2(phi).
	// IMPROVEMENT: Increased safety margin to +128 bits for robustness.
	prec := uint(float64(n)*log2Phi + 128)

	// Initialize high-precision floats
	phiPrec := new(big.Float).SetPrec(prec).Set(phi)
	sqrt5Prec := new(big.Float).SetPrec(prec).Set(sqrt5)

	// Calculate phi^n using binary exponentiation.
	result := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)
	exponent := uint(n)

	numBitsInN := bits.Len(uint(n))
	currentStep := 0

	for exponent > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exponent&1 == 1 {
			result.Mul(result, base)
		}

		if exponent > 1 {
			base.Mul(base, base)
		}
		exponent >>= 1

		currentStep++
		if progress != nil && numBitsInN > 0 {
			// Cap progress slightly as the final division/rounding is fast.
			progress <- (float64(currentStep) / float64(numBitsInN)) * 99.0
		}
	}

	// (phi^n) / sqrt(5)
	result.Quo(result, sqrt5Prec)

	// Round to the nearest integer: floor(result + 0.5)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	result.Add(result, half)

	z := new(big.Int)
	result.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// ------------------------------------------------------------
// Algorithm 4: Optimized Iterative (O(N))
// ------------------------------------------------------------

// fibIterative provides a baseline O(N) implementation using the pool.
func fibIterative(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if res, err := handleBaseCases(n, progress); res != nil || err != nil {
		return res, err
	}

	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	// Use a temporary variable from the pool for the swap.
	temp := pool.Get().(*big.Int)
	defer pool.Put(temp)

	// Define reporting interval to avoid overwhelming the progress channel.
	reportInterval := 1000
	if n > 100000 {
		reportInterval = n / 100 // Report approx 100 times total
	}

	for i := 2; i <= n; i++ {
		// Periodic cancellation check and progress reporting.
		if i%reportInterval == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				if progress != nil {
					progress <- (float64(i) / float64(n)) * 100.0
				}
			}
		}

		// temp = a + b; a = b; b = temp
		temp.Add(a, b)
		a.Set(b)
		b.Set(temp)
	}

	if progress != nil {
		progress <- 100.0
	}

	// Return a copy of 'b'.
	return new(big.Int).Set(b), nil
}

// ------------------------------------------------------------
// Main Execution Logic (Refactored)
// ------------------------------------------------------------

func main() {
	// 1. Parse Flags
	nFlag := flag.Int("n", 2500000, "Index n of the Fibonacci term. Default is 100,000.")
	timeoutFlag := flag.Duration("timeout", 2*time.Minute, "Global maximum execution time (e.g., 1m, 30s).")
	runAlgosFlag := flag.String("runAlgos", "all", "Comma-separated list of algorithms (e.g., 'binet,fast').")
	flag.BoolVar(&verboseFlag, "v", false, "Verbose output: display the full Fibonacci number.")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag
	runAlgosStr := strings.ToLower(*runAlgosFlag)

	if n < 0 {
		log.Fatalf("Index n must be >= 0. Received: %d", n)
	}

	// 2. Define and Select Tasks
	definedTasks := []task{
		{"Fast-Doubling", fibFastDoubling},
		{"Matrix 2x2", fibMatrix},
		{"Binet (Float)", fibBinet},
		{"Iterative (O(N))", fibIterative},
	}

	selectedTasks, selectedAlgoNames := selectTasks(definedTasks, runAlgosStr)

	if len(selectedTasks) == 0 {
		log.Println("No valid algorithms selected. Exiting.")
		return
	}

	log.Printf("Calculating F(%d) with a timeout of %v...", n, timeout)
	log.Printf("Algorithms: %s", strings.Join(selectedAlgoNames, ", "))

	// 3. Setup Context, Pool, and Channels
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	intPool := newIntPool()
	progressAggregatorCh := make(chan progressData, len(selectedTasks)*50)
	resultsCh := make(chan result, len(selectedTasks))

	// 4. Start Progress Printer
	var printerWg sync.WaitGroup
	printerWg.Add(1)
	go func() {
		defer printerWg.Done()
		// Pass the context to the printer
		progressPrinter(ctx, progressAggregatorCh, selectedAlgoNames)
	}()

	// 5. Launch Concurrent Calculations
	var calculationWg sync.WaitGroup
	var relayWg sync.WaitGroup

	log.Println("Launching concurrent calculations...")

	for _, t := range selectedTasks {
		calculationWg.Add(1)
		go runTask(ctx, t, n, intPool, &calculationWg, &relayWg, progressAggregatorCh, resultsCh)
	}

	// 6. Wait for Completion
	calculationWg.Wait()
	// Wait for relays to finish draining before closing the aggregator channel.
	relayWg.Wait()
	close(progressAggregatorCh)
	// Wait for the printer to display the final status.
	printerWg.Wait()

	// 7. Process Results
	processResults(resultsCh, selectedTasks, n)

	log.Println("Program finished.")
}

// selectTasks parses the user input string with normalization for robust matching.
func selectTasks(definedTasks []task, runAlgosStr string) ([]task, []string) {
	// Normalize names for flexible matching
	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		// Remove descriptors for easier matching (e.g. "binet" matches "Binet (Float)")
		s = strings.ReplaceAll(s, "(float)", "")
		s = strings.ReplaceAll(s, "(o(n))", "")
		s = strings.ReplaceAll(s, "2x2", "")
		return s
	}

	availableAlgos := make(map[string]task)
	for _, t := range definedTasks {
		availableAlgos[normalize(t.name)] = t
	}

	var selectedTasks []task
	var selectedAlgoNames []string

	if runAlgosStr == "all" || runAlgosStr == "" {
		for _, t := range definedTasks {
			selectedTasks = append(selectedTasks, t)
			selectedAlgoNames = append(selectedAlgoNames, t.name)
		}
		return selectedTasks, selectedAlgoNames
	}

	userRequestedAlgoNames := strings.Split(runAlgosStr, ",")
	addedAlgos := make(map[string]bool)

	for _, name := range userRequestedAlgoNames {
		normalizedName := normalize(strings.TrimSpace(name))
		found := false

		// Allow partial matching (e.g., "fast" matches "fastdoubling")
		for key, task := range availableAlgos {
			if strings.Contains(key, normalizedName) && normalizedName != "" {
				if !addedAlgos[task.name] {
					selectedTasks = append(selectedTasks, task)
					selectedAlgoNames = append(selectedAlgoNames, task.name)
					addedAlgos[task.name] = true
					found = true
				}
				break
			}
		}

		if !found {
			log.Printf("Warning: Algorithm '%s' not recognized or matched.", name)
		}
	}
	return selectedTasks, selectedAlgoNames
}

// runTask executes a specific Fibonacci task and manages its lifecycle, including progress relay.
func runTask(ctx context.Context, currentTask task, n int, pool *sync.Pool, calculationWg *sync.WaitGroup, relayWg *sync.WaitGroup, progressAggregatorCh chan<- progressData, resultsCh chan<- result) {
	defer calculationWg.Done()

	localProgCh := make(chan float64, 10)

	// Goroutine to relay progress to the central aggregator.
	relayWg.Add(1)
	go func() {
		defer relayWg.Done()
		for p := range localProgCh {
			select {
			case progressAggregatorCh <- progressData{currentTask.name, p}:
			case <-ctx.Done():
				return
			}
		}
	}()

	start := time.Now()
	v, err := currentTask.fn(ctx, localProgCh, n, pool)
	duration := time.Since(start)
	close(localProgCh)

	resultsCh <- result{currentTask.name, v, duration, err}
}

// processResults collects, sorts, and displays the results.
func processResults(resultsCh chan result, selectedTasks []task, n int) {
	results := make([]result, 0, len(selectedTasks))

	// Collect results
	for i := 0; i < len(selectedTasks); i++ {
		r := <-resultsCh
		results = append(results, r)
		// Log errors immediately upon reception
		if r.err != nil {
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				log.Printf("⚠️ Task '%s' timed out or was cancelled after %v", r.name, r.duration.Round(time.Microsecond))
			} else {
				log.Printf("❌ Error for task '%s': %v (duration: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
	}
	close(resultsCh)

	// Sort results: successful ones first, then by duration.
	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		return results[i].duration < results[j].duration
	})

	// Display Ordered Results
	fmt.Println("\n--------------------------- ORDERED RESULTS ---------------------------")
	for _, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			status = "Error"
			if r.err == context.DeadlineExceeded || r.err == context.Canceled {
				status = "Timeout/Canceled"
			}
		} else if r.value != nil {
			valStr = summarizeBigInt(r.value)
		}
		fmt.Printf("%-18s : %-15v [%-16s] Result: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	// Verification and Detailed Output
	fmt.Println("\n--------------------------- VERIFICATION ------------------------------")
	verifyAndPrintDetails(results, n)
}

// summarizeBigInt provides a short representation of a large number.
func summarizeBigInt(v *big.Int) string {
	s := v.String()
	if len(s) > 15 {
		return s[:5] + "..." + s[len(s)-5:]
	}
	return s
}

// verifyAndPrintDetails compares results and prints details of the fastest successful one.
func verifyAndPrintDetails(results []result, n int) {
	var firstSuccessfulResult *result
	allValidResultsIdentical := true
	successfulCount := 0

	for i := range results {
		if results[i].err == nil && results[i].value != nil {
			successfulCount++
			if firstSuccessfulResult == nil {
				firstSuccessfulResult = &results[i]
				fmt.Printf("Fastest successful algorithm: %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
				printFibResultDetails(firstSuccessfulResult.value, n)
			} else {
				if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
					allValidResultsIdentical = false
					log.Printf("⚠️ DISCREPANCY DETECTED! Result of '%s' differs from '%s'.",
						results[i].name, firstSuccessfulResult.name)
				}
			}
		}
	}

	if successfulCount > 0 {
		if allValidResultsIdentical {
			fmt.Println("✅ Verification successful: All completed algorithms yielded identical results.")
		} else {
			fmt.Println("❌ Verification failed: Algorithm results diverge!")
		}
	} else {
		fmt.Println("❌ No algorithm successfully completed the calculation.")
	}
}

// printFibResultDetails displays detailed information about the calculated Fibonacci number.
func printFibResultDetails(value *big.Int, n int) {
	if value == nil {
		return
	}
	s := value.Text(10)
	digits := len(s)
	fmt.Printf("Number of digits in F(%d): %d\n", n, digits)

	if verboseFlag {
		fmt.Printf("Value = %s\n", s)
	} else if digits > 50 {
		// Display scientific notation and the start/end.
		floatVal := new(big.Float).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Value ≈ %s\n", sci)
		fmt.Printf("Value = %s...%s\n", s[:10], s[len(s)-10:])
	} else {
		fmt.Printf("Value = %s\n", s)
	}
}
