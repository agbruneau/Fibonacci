package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/bits"
	"sort"
	"strings" // Added missing import
	"sync"
	"time"
)

// main.go
//
// This program calculates the nth Fibonacci number using three different algorithms:
// 1. Binet's formula (using big.Float for precision).
// 2. Fast Doubling algorithm.
// 3. Matrix exponentiation algorithm (2x2 Matrix).
//
// It executes these algorithms concurrently, displays their progress,
// and compares their execution times and results.
// A sync.Pool is used to reduce memory allocations for big.Int objects.
//
// Usage:
//   go run main.go -n <index> -timeout <duration>
// Example:
//   go run main.go -n 100000 -timeout 1m

// ------------------------------------------------------------
// Optimized types and structures
// ------------------------------------------------------------

// fibFunc is a type for functions calculating Fibonacci numbers.
// It takes a context for cancellation, a channel for progress,
// the index n, and a pool of big.Int objects for memory reuse.
type fibFunc func(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error)

// task represents a Fibonacci calculation task to be executed.
type task struct {
	name string  // Algorithm name
	fn   fibFunc // Algorithm function
}

// result stores the result of a calculation task.
type result struct {
	name     string        // Algorithm name
	value    *big.Int      // Calculated Fibonacci value
	duration time.Duration // Calculation duration
	err      error         // Potential error
}

// ------------------------------------------------------------
// Precomputed constants for Binet (used as a base for dynamic precision)
// ------------------------------------------------------------
var (
	// phi and sqrt5 are placeholders. Their values are computed with
	// sufficient precision in fibBinet.
	phi   = big.NewFloat(0)
	sqrt5 = big.NewFloat(0)
)

// ------------------------------------------------------------
// Progress display management
// ------------------------------------------------------------

// progressData encapsulates progress information for a task.
type progressData struct {
	name string  // Task name
	pct  float64 // Progress percentage
}

// progressPrinter manages the consolidated display of progress for all tasks.
// It refreshes the display at regular intervals or upon new progress data.
// `taskNames` is the list of names of tasks that will be executed, for display purposes.
func progressPrinter(progress <-chan progressData, taskNames []string) {
	status := make(map[string]float64)
	ticker := time.NewTicker(100 * time.Millisecond) // Minimal refresh for the interface
	defer ticker.Stop()
	lastPrintTime := time.Now()
	needsUpdate := true

	// Use the passed taskNames to initialize and order the display.
	// It's important that these names correspond to the task.name of the executed tasks.
	// Sorting here ensures a constant display if the original order of taskNames is not guaranteed.
	// However, if taskNames comes from selectedTasks, the order will be that of definition.
	// For consistency, taskNames can be sorted.
	// sort.Strings(taskNames) // Optional, if alphabetical order is preferred.

	for _, k := range taskNames {
		status[k] = 0.0
	}

	// If no tasks are executed, we can return early.
	if len(taskNames) == 0 {
		return
	}

	for {
		select {
		case p, ok := <-progress:
			if !ok { // Channel closed, end of progress
				printStatus(status, taskNames)
				fmt.Println() // Final newline
				return
			}
			if _, exists := status[p.name]; exists { // Only update if the task is being tracked
				if status[p.name] != p.pct {
					status[p.name] = p.pct
					needsUpdate = true
				}
			}
		case <-ticker.C:
			// Refresh if an update is needed or if a certain time has passed,
			// to show that the program is still active even if percentages don't change quickly.
			if needsUpdate || time.Since(lastPrintTime) > 500*time.Millisecond {
				printStatus(status, taskNames)
				needsUpdate = false // Reset after printing
				lastPrintTime = time.Now()
			}
		}
	}
}

// printStatus displays the current progress status for each task.
func printStatus(status map[string]float64, keys []string) {
	fmt.Print("\r") // Carriage return to clear the previous line
	first := true
	for _, k := range keys {
		v, ok := status[k]
		if !ok {
			continue
		} // In case a task hasn't sent status yet
		if !first {
			fmt.Print("   ")
		}
		fmt.Printf("%-14s %6.2f%%", k, v)
		first = false
	}
	fmt.Print("                 ") // Spaces to clear any remnants of the previous line
}

// ------------------------------------------------------------
// Pool of big.Int for memory reuse
// ------------------------------------------------------------

// newIntPool creates a new sync.Pool for *big.Int objects.
// This helps reduce pressure on the garbage collector by reusing objects.
func newIntPool() *sync.Pool {
	return &sync.Pool{
		New: func() interface{} {
			return new(big.Int)
		},
	}
}

// fibTempInts is a structure to manage a set of temporary *big.Int
// used in Fibonacci calculations, to simplify pool management.
type fibTempInts struct {
	a_orig *big.Int
	t1     *big.Int
	t2     *big.Int
	t3     *big.Int
	new_a  *big.Int
	new_b  *big.Int
	// t_sum is used in the 'if (uint(n)>>i)&1 == 1' branch
	// It is included here to centralize the management of temporary variables.
	t_sum *big.Int
}

// acquire retrieves all necessary *big.Int from the pool.
func (tmp *fibTempInts) acquire(pool *sync.Pool) {
	tmp.a_orig = pool.Get().(*big.Int)
	tmp.t1 = pool.Get().(*big.Int)
	tmp.t2 = pool.Get().(*big.Int)
	tmp.t3 = pool.Get().(*big.Int)
	tmp.new_a = pool.Get().(*big.Int)
	tmp.new_b = pool.Get().(*big.Int)
	tmp.t_sum = pool.Get().(*big.Int)
}

// release returns all *big.Int to the pool.
// It is crucial to call this method to avoid memory leaks from the pool.
func (tmp *fibTempInts) release(pool *sync.Pool) {
	pool.Put(tmp.a_orig)
	pool.Put(tmp.t1)
	pool.Put(tmp.t2)
	pool.Put(tmp.t3)
	pool.Put(tmp.new_a)
	pool.Put(tmp.new_b)
	pool.Put(tmp.t_sum)
}

// ------------------------------------------------------------
// Fibonacci calculation algorithms
// ------------------------------------------------------------

// fibBinet calculates F(n) using Binet's formula.
// F(n) = (phi^n - (-phi)^-n) / sqrt(5)
// For large n, this simplifies to F(n) ≈ round(phi^n / sqrt(5)).
// Note: This algorithm uses big.Float and therefore does not actively use the big.Int pool,
// as most allocations concern high-precision floats.
// The pool is passed for consistency with the fibFunc signature.
func fibBinet(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative index not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	// The required precision increases with n.
	// bits for phi^n ≈ n * log2(phi)
	// Add a safety margin (+10 or more) for precision.
	phiVal := (1 + math.Sqrt(5)) / 2 // Used only for precision estimation
	prec := uint(float64(n)*math.Log2(phiVal) + 10)

	// Compute sqrt(5) with the requested precision.
	sqrt5Prec := new(big.Float).SetPrec(prec).SetFloat64(5)
	sqrt5Prec.Sqrt(sqrt5Prec)

	// Compute phi = (1 + sqrt(5)) / 2 with the same precision.
	phiPrec := new(big.Float).SetPrec(prec)
	phiPrec.Add(sqrt5Prec, new(big.Float).SetPrec(prec).SetFloat64(1))
	phiPrec.Quo(phiPrec, new(big.Float).SetPrec(prec).SetFloat64(2))

	// Calculate phi^n by binary exponentiation (exponentiation by squaring)
	// to minimize the number of multiplications.
	powN := new(big.Float).SetPrec(prec)

	numBitsInN := bits.Len(uint(n))
	currentStep := 0

	phiToN := new(big.Float).SetPrec(prec).SetInt64(1)
	base := new(big.Float).SetPrec(prec).Set(phiPrec)

	exponent := uint(n)
	for exponent > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if exponent&1 == 1 {
			phiToN.Mul(phiToN, base)
		}
		base.Mul(base, base)
		exponent >>= 1

		currentStep++
		if progress != nil && numBitsInN > 0 {
			progress <- (float64(currentStep) / float64(numBitsInN)) * 100.0
		}
	}
	powN = phiToN

	powN.Quo(powN, sqrt5Prec)

	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)
	powN.Add(powN, half)

	z := new(big.Int)
	powN.Int(z)

	if progress != nil {
		progress <- 100.0
	}
	return z, nil
}

// fibFastDoubling calculates F(n) using the "Fast Doubling" algorithm.
// Formulas:
// F(2k) = F(k) * [2*F(k+1) – F(k)]
// F(2k+1) = F(k)^2 + F(k+1)^2
// This algorithm is very efficient and uses the big.Int pool for intermediate calculations.
func fibFastDoubling(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative index not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	// a and b represent F(k) and F(k+1)
	a := pool.Get().(*big.Int).SetInt64(0)
	defer pool.Put(a)
	b := pool.Get().(*big.Int).SetInt64(1)
	defer pool.Put(b)

	totalBits := bits.Len(uint(n))

	// Initialize the structure for temporary variables
	temps := fibTempInts{}

	for i := totalBits - 1; i >= 0; i-- {
		temps.acquire(pool) // Acquire temporary variables at the start of each iteration

		select {
		case <-ctx.Done():
			temps.release(pool) // Ensure release in case of cancellation
			return nil, ctx.Err()
		default:
		}

		// a_orig stores the value of 'a' (F(k)) before its modification
		temps.a_orig.Set(a)

		// Calculate F(2k) = a * (2*b - a)
		// temps.t1 = 2*b
		temps.t1.Lsh(b, 1)
		// temps.t1 = 2*b - a_orig
		temps.t1.Sub(temps.t1, temps.a_orig)
		// temps.new_a = a_orig * temps.t1
		temps.new_a.Mul(temps.a_orig, temps.t1)

		// Calculate F(2k+1) = a^2 + b^2
		// temps.t2 = a_orig^2
		temps.t2.Mul(temps.a_orig, temps.a_orig)
		// temps.t3 = b^2
		temps.t3.Mul(b, b)
		// temps.new_b = temps.t2 + temps.t3
		temps.new_b.Add(temps.t2, temps.t3)

		// Update a and b with the new values F(2k) and F(2k+1)
		a.Set(temps.new_a)
		b.Set(temps.new_b)

		if (uint(n)>>i)&1 == 1 { // If the i-th bit of n is 1 (n is odd at this step)
			// We have F(2k) and F(2k+1). We want F(2k+1) and F(2k+2).
			// F(2k+1) is in 'b' (temps.new_b after the update)
			// F(2k+2) = F(2k) + F(2k+1)
			// 'a' contains F(2k) (temps.new_a) and 'b' contains F(2k+1) (temps.new_b)
			// temps.t_sum = a + b
			temps.t_sum.Add(a, b)
			// a becomes F(2k+1)
			a.Set(b)
			// b becomes F(2k+2)
			b.Set(temps.t_sum)
		}

		temps.release(pool) // Release temporary variables at the end of the iteration

		if progress != nil && totalBits > 0 {
			progress <- (float64(totalBits-i) / float64(totalBits)) * 100.0
		}
	}

	if progress != nil {
		progress <- 100.0
	}

	// 'a' contains F(n). Create a copy for the return as 'a' belongs to the pool.
	finalResult := new(big.Int).Set(a)
	return finalResult, nil
}

// mat2 represents a 2x2 matrix of *big.Int.
type mat2 struct {
	a, b, c, d *big.Int
}

// newMat2 creates a new mat2 whose components are from the pool.
func newMat2(pool *sync.Pool) *mat2 {
	return &mat2{
		a: pool.Get().(*big.Int),
		b: pool.Get().(*big.Int),
		c: pool.Get().(*big.Int),
		d: pool.Get().(*big.Int),
	}
}

// release returns the matrix components to the pool.
func (m *mat2) release(pool *sync.Pool) {
	pool.Put(m.a)
	pool.Put(m.b)
	pool.Put(m.c)
	pool.Put(m.d)
}

// set updates the values of the target matrix with those of another matrix.
func (m *mat2) set(other *mat2) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// matMul performs the multiplication of two matrices m1 * m2 and stores the result in target.
// target must not be aliased with m1 or m2 if an operation like target = target * m1 is performed.
// Use a temporary matrix in that case.
func matMul(target, m1, m2 *mat2, pool *sync.Pool) {
	t1 := pool.Get().(*big.Int)
	t2 := pool.Get().(*big.Int)
	defer pool.Put(t1)
	defer pool.Put(t2)

	// Calculate target.a = (m1.a*m2.a) + (m1.b*m2.c)
	t1.Mul(m1.a, m2.a)
	t2.Mul(m1.b, m2.c)
	target.a.Add(t1, t2)

	// Calculate target.b = (m1.a*m2.b) + (m1.b*m2.d)
	t1.Mul(m1.a, m2.b)
	t2.Mul(m1.b, m2.d)
	target.b.Add(t1, t2)

	// Calculate target.c = (m1.c*m2.a) + (m1.d*m2.c)
	t1.Mul(m1.c, m2.a)
	t2.Mul(m1.d, m2.c)
	target.c.Add(t1, t2)

	// Calculate target.d = (m1.c*m2.b) + (m1.d*m2.d)
	t1.Mul(m1.c, m2.b)
	t2.Mul(m1.d, m2.d)
	target.d.Add(t1, t2)
}

// fibMatrix calculates F(n) by exponentiation of the matrix [[1,1],[1,0]].
func fibMatrix(ctx context.Context, progress chan<- float64, n int, pool *sync.Pool) (*big.Int, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative index not supported: %d", n)
	}
	if n <= 1 {
		if progress != nil {
			progress <- 100.0
		}
		return big.NewInt(int64(n)), nil
	}

	res := newMat2(pool)
	defer res.release(pool)
	res.a.SetInt64(1)
	res.b.SetInt64(0)
	res.c.SetInt64(0)
	res.d.SetInt64(1) // Identity matrix

	base := newMat2(pool)
	defer base.release(pool)
	base.a.SetInt64(1)
	base.b.SetInt64(1)
	base.c.SetInt64(1)
	base.d.SetInt64(0) // Fibonacci matrix

	temp := newMat2(pool) // Temporary matrix for calculations
	defer temp.release(pool)

	exp := uint(n - 1)
	totalSteps := bits.Len(exp)
	stepsDone := 0

	for ; exp > 0; exp >>= 1 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // Defers will handle the release
		default:
		}
		if exp&1 == 1 {
			// res = res * base
			matMul(temp, res, base, pool)
			res.set(temp)
		}
		if exp > 1 { // Condition to avoid unnecessary multiplication at the last step
			// base = base * base
			matMul(temp, base, base, pool)
			base.set(temp)
		}

		stepsDone++
		if progress != nil && totalSteps > 0 {
			progress <- (float64(stepsDone) / float64(totalSteps)) * 100.0
		}
	}

	if progress != nil {
		progress <- 100.0
	}
	// The result is copied to avoid returning a *big.Int that is part of a pooled matrix.
	return new(big.Int).Set(res.a), nil
}

// ------------------------------------------------------------
// Main function
// ------------------------------------------------------------
func main() {
	nFlag := flag.Int("n", 10000000, "Index n of the Fibonacci term (non-negative integer)")
	timeoutFlag := flag.Duration("timeout", 1*time.Minute, "Global maximum execution time")
	runAlgosFlag := flag.String("runAlgos", "all", "Comma-separated list of algorithms to run (e.g., 'binet,fast-doubling'). 'all' runs every algorithm. Names are case-insensitive. Available: fast-doubling, binet, matrice 2x2.")
	flag.Parse()

	n := *nFlag
	timeout := *timeoutFlag
	runAlgosStr := strings.ToLower(*runAlgosFlag)

	if n < 0 {
		log.Fatalf("Index n must be greater than or equal to 0. Received: %d", n)
	}

	// definedTasks preserves the original order for "all" and provides a canonical list.
	definedTasks := []task{
		{"Fast-doubling", fibFastDoubling},
		{"Binet", fibBinet},
		{"Matrice 2x2", fibMatrix},
	}

	availableAlgos := make(map[string]task)
	for _, t := range definedTasks {
		availableAlgos[strings.ToLower(t.name)] = t
	}

	var selectedTasks []task
	var selectedAlgoNames []string // For the progressPrinter

	if runAlgosStr == "all" || runAlgosStr == "" {
		selectedTasks = definedTasks // Use the original ordered slice
		for _, t := range selectedTasks {
			selectedAlgoNames = append(selectedAlgoNames, t.name)
		}
	} else {
		userRequestedAlgoNames := strings.Split(runAlgosStr, ",")
		addedAlgos := make(map[string]bool) // To prevent duplicates if user lists an algo multiple times

		for _, name := range userRequestedAlgoNames {
			trimmedName := strings.TrimSpace(name)
			lowerName := strings.ToLower(trimmedName)
			if task, found := availableAlgos[lowerName]; found {
				if !addedAlgos[lowerName] {
					selectedTasks = append(selectedTasks, task)
					selectedAlgoNames = append(selectedAlgoNames, task.name) // Use original casing from task.name
					addedAlgos[lowerName] = true
				}
			} else {
				log.Printf("Warning: Algorithm '%s' not recognized and will be skipped.", trimmedName)
			}
		}
	}

	if len(selectedTasks) == 0 {
		log.Println("No valid algorithms selected or no algorithms matching the provided names. Please check the -runAlgos flag.")
		log.Printf("Available algorithms: Fast-doubling, Binet, Matrice 2x2")
		return
	}

	log.Printf("Calculating F(%d) with a timeout of %v...\n", n, timeout)
	log.Printf("Algorithms to run: %s\n", strings.Join(selectedAlgoNames, ", "))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	intPool := newIntPool()

	progressAggregatorCh := make(chan progressData, len(selectedTasks)*2)
	go progressPrinter(progressAggregatorCh, selectedAlgoNames) // Pass the names of selected tasks

	var wg sync.WaitGroup
	var relayWg sync.WaitGroup
	resultsCh := make(chan result, len(selectedTasks))

	log.Println("Launching concurrent calculations...")

	for _, t := range selectedTasks { // Use selectedTasks here
		wg.Add(1)
		go func(currentTask task) {
			defer wg.Done()

			localProgCh := make(chan float64, 10)

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
			v, err := currentTask.fn(ctx, localProgCh, n, intPool)
			duration := time.Since(start)
			close(localProgCh)

			resultsCh <- result{currentTask.name, v, duration, err}
		}(t)
	}

	wg.Wait()
	log.Println("Calculation goroutines finished.")

	relayWg.Wait()
	log.Println("Progress relay goroutines finished.")

	close(progressAggregatorCh)

	// Retrieve results for the number of selected tasks
	results := make([]result, 0, len(selectedTasks))
	timeoutOccurredOverall := false
	if ctx.Err() == context.DeadlineExceeded {
		timeoutOccurredOverall = true
	}

	for i := 0; i < len(selectedTasks); i++ { // Use len(selectedTasks) here
		r := <-resultsCh
		if r.err != nil {
			if r.err == context.DeadlineExceeded {
				log.Printf("⚠️ Task '%s' interrupted by timeout (or context canceled) after %v", r.name, r.duration.Round(time.Microsecond))
			} else {
				log.Printf("❌ Error for task '%s': %v (duration: %v)", r.name, r.err, r.duration.Round(time.Microsecond))
			}
		}
		results = append(results, r)
	}
	close(resultsCh)

	if timeoutOccurredOverall {
		log.Println("Comparison of results and final details may be affected because the global timeout was reached.")
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].err == nil && results[j].err != nil {
			return true
		}
		if results[i].err != nil && results[j].err == nil {
			return false
		}
		return results[i].duration < results[j].duration
	})

	fmt.Println("\n--------------------------- ORDERED RESULTS ---------------------------")
	for _, r := range results {
		status := "OK"
		valStr := "N/A"
		if r.err != nil {
			status = fmt.Sprintf("Error: %v", r.err)
			if r.err == context.DeadlineExceeded {
				status = "Timeout/Canceled"
			}
		} else if r.value != nil {
			if len(r.value.String()) > 15 {
				valStr = r.value.String()[:5] + "..." + r.value.String()[len(r.value.String())-5:]
			} else {
				valStr = r.value.String()
			}
		}
		fmt.Printf("%-15s : %-12v [%-14s] Result: %s\n", r.name, r.duration.Round(time.Microsecond), status, valStr)
	}

	if len(results) > 0 { // Only if results were produced
		var firstSuccessfulResult *result
		allValidResultsIdentical := true
		foundSuccessful := false

		for i := range results {
			if results[i].err == nil && results[i].value != nil {
				if !foundSuccessful {
					firstSuccessfulResult = &results[i]
					foundSuccessful = true
					fmt.Printf("\nFastest algorithm (that succeeded): %s (%v)\n", firstSuccessfulResult.name, firstSuccessfulResult.duration.Round(time.Microsecond))
					printFibResultDetails(firstSuccessfulResult.value, n, firstSuccessfulResult.duration)
				} else {
					if results[i].value.Cmp(firstSuccessfulResult.value) != 0 {
						allValidResultsIdentical = false
						// Assuming Go 1.21+ for built-in min function.
						log.Printf("⚠️ DISCREPANCY! Result of '%s' (%s...) different from '%s' (%s...)",
							results[i].name, results[i].value.String()[:min(10, len(results[i].value.String()))], // Use built-in min
							firstSuccessfulResult.name, firstSuccessfulResult.value.String()[:min(10, len(firstSuccessfulResult.value.String()))]) // Use built-in min
					}
				}
			}
		}

		if foundSuccessful {
			if allValidResultsIdentical {
				fmt.Println("✅ All valid results produced are identical.")
			} else {
				fmt.Println("❌ Valid algorithm results diverge!")
			}
		} else {
			fmt.Println("\nNo algorithm successfully completed to produce a result among those selected.")
		}
	} else if runAlgosStr != "all" && runAlgosStr != "" {
		// Message if specific algos were requested but none could be launched (already handled by the 'return' earlier)
		// This block is potentially redundant with the check for len(selectedTasks) == 0 above.
	} else if len(definedTasks) > 0 && len(selectedTasks) == 0 {
		// Case where "all" is implicit but selectedTasks is empty (should not happen if allTasks is not empty)
		log.Println("No algorithm was executed.")
	}

	log.Println("Program finished.")
}

// printFibResultDetails displays detailed information about the calculated Fibonacci number.
func printFibResultDetails(value *big.Int, n int, duration time.Duration) {
	if value == nil {
		return
	}
	digits := len(value.Text(10))
	fmt.Printf("F(%d) calculated in %v\n", n, duration.Round(time.Millisecond))
	fmt.Printf("Number of digits: %d\n", digits)

	if digits > 20 {
		floatVal := new(big.Float).SetPrec(uint(digits + 10)).SetInt(value)
		sci := floatVal.Text('e', 8)
		fmt.Printf("Value of F(%d) ≈ %s (scientific notation)\n", n, sci)
	} else {
		fmt.Printf("Value of F(%d) = %s\n", n, value.Text(10))
	}
}

// Note: The custom 'min' function has been removed.
// It's assumed that the code will be compiled with Go 1.21+ which includes a built-in 'min' function.
// If using an older Go version, you would need to retain the custom 'min' function.
