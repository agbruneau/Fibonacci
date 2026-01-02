package fibonacci

import (
	"math/big"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// checkLimit Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestCheckLimit tests the pool limit checker function.
func TestCheckLimit(t *testing.T) {
	t.Parallel()

	t.Run("nil returns false", func(t *testing.T) {
		t.Parallel()
		if checkLimit(nil) {
			t.Error("checkLimit(nil) = true, want false")
		}
	})

	t.Run("small number returns false", func(t *testing.T) {
		t.Parallel()
		small := big.NewInt(12345)
		if checkLimit(small) {
			t.Errorf("checkLimit(small with %d bits) = true, want false", small.BitLen())
		}
	})

	t.Run("exactly at limit returns false", func(t *testing.T) {
		t.Parallel()
		// Create a number with exactly MaxPooledBitLen bits
		atLimit := new(big.Int).Lsh(big.NewInt(1), uint(MaxPooledBitLen-1))
		if checkLimit(atLimit) {
			t.Errorf("checkLimit(at limit with %d bits) = true, want false", atLimit.BitLen())
		}
	})

	t.Run("above limit returns true", func(t *testing.T) {
		t.Parallel()
		// Create a number exceeding MaxPooledBitLen
		aboveLimit := new(big.Int).Lsh(big.NewInt(1), uint(MaxPooledBitLen+1))
		if !checkLimit(aboveLimit) {
			t.Errorf("checkLimit(above limit with %d bits) = false, want true", aboveLimit.BitLen())
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Task Execution Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestExecuteTasksSequential tests sequential task execution.
func TestExecuteTasksSequential(t *testing.T) {
	t.Parallel()

	x := big.NewInt(100)
	y := big.NewInt(200)
	expected := new(big.Int).Mul(x, y)

	var result *big.Int
	tasks := []multiplicationTask{
		{
			dest:         &result,
			a:            x,
			b:            y,
			fftThreshold: 0,
		},
	}

	err := executeTasks[multiplicationTask, *multiplicationTask](tasks, false)
	if err != nil {
		t.Fatalf("executeTasks failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("result = %s, want %s", result.String(), expected.String())
	}
}

// TestExecuteTasksParallel tests parallel task execution.
func TestExecuteTasksParallel(t *testing.T) {
	t.Parallel()

	// Create multiple multiplication tasks
	var results [3]*big.Int
	tasks := []multiplicationTask{
		{dest: &results[0], a: big.NewInt(10), b: big.NewInt(20), fftThreshold: 0},
		{dest: &results[1], a: big.NewInt(30), b: big.NewInt(40), fftThreshold: 0},
		{dest: &results[2], a: big.NewInt(50), b: big.NewInt(60), fftThreshold: 0},
	}

	expectedResults := []*big.Int{
		big.NewInt(200),
		big.NewInt(1200),
		big.NewInt(3000),
	}

	err := executeTasks[multiplicationTask, *multiplicationTask](tasks, true)
	if err != nil {
		t.Fatalf("executeTasks parallel failed: %v", err)
	}

	for i, expected := range expectedResults {
		if results[i] == nil {
			t.Errorf("results[%d] is nil", i)
			continue
		}
		if results[i].Cmp(expected) != 0 {
			t.Errorf("results[%d] = %s, want %s", i, results[i].String(), expected.String())
		}
	}
}

// TestExecuteSquaringTasks tests squaring task execution.
func TestExecuteSquaringTasks(t *testing.T) {
	t.Parallel()

	x := big.NewInt(123)
	expected := new(big.Int).Mul(x, x)

	var result *big.Int
	tasks := []squaringTask{
		{
			dest:         &result,
			x:            x,
			fftThreshold: 0,
		},
	}

	err := executeTasks[squaringTask, *squaringTask](tasks, false)
	if err != nil {
		t.Fatalf("executeTasks failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Cmp(expected) != 0 {
		t.Errorf("result = %s, want %s", result.String(), expected.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// executeMixedTasks Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestExecuteMixedTasksEmpty tests empty task slices.
func TestExecuteMixedTasksEmpty(t *testing.T) {
	t.Parallel()

	err := executeMixedTasks(nil, nil, false)
	if err != nil {
		t.Errorf("executeMixedTasks with empty slices failed: %v", err)
	}

	err = executeMixedTasks(nil, nil, true)
	if err != nil {
		t.Errorf("executeMixedTasks parallel with empty slices failed: %v", err)
	}
}

// TestExecuteMixedTasksSequential tests sequential mixed task execution.
func TestExecuteMixedTasksSequential(t *testing.T) {
	t.Parallel()

	var sqrResult, mulResult *big.Int

	sqrTasks := []squaringTask{
		{dest: &sqrResult, x: big.NewInt(10), fftThreshold: 0},
	}
	mulTasks := []multiplicationTask{
		{dest: &mulResult, a: big.NewInt(5), b: big.NewInt(6), fftThreshold: 0},
	}

	err := executeMixedTasks(sqrTasks, mulTasks, false)
	if err != nil {
		t.Fatalf("executeMixedTasks failed: %v", err)
	}

	if sqrResult == nil || sqrResult.Cmp(big.NewInt(100)) != 0 {
		t.Errorf("sqrResult = %v, want 100", sqrResult)
	}
	if mulResult == nil || mulResult.Cmp(big.NewInt(30)) != 0 {
		t.Errorf("mulResult = %v, want 30", mulResult)
	}
}

// TestExecuteMixedTasksParallel tests parallel mixed task execution.
func TestExecuteMixedTasksParallel(t *testing.T) {
	t.Parallel()

	var sqrResults [2]*big.Int
	var mulResults [2]*big.Int

	sqrTasks := []squaringTask{
		{dest: &sqrResults[0], x: big.NewInt(10), fftThreshold: 0},
		{dest: &sqrResults[1], x: big.NewInt(20), fftThreshold: 0},
	}
	mulTasks := []multiplicationTask{
		{dest: &mulResults[0], a: big.NewInt(3), b: big.NewInt(4), fftThreshold: 0},
		{dest: &mulResults[1], a: big.NewInt(5), b: big.NewInt(6), fftThreshold: 0},
	}

	err := executeMixedTasks(sqrTasks, mulTasks, true)
	if err != nil {
		t.Fatalf("executeMixedTasks parallel failed: %v", err)
	}

	// Verify squaring results
	if sqrResults[0] == nil || sqrResults[0].Cmp(big.NewInt(100)) != 0 {
		t.Errorf("sqrResults[0] = %v, want 100", sqrResults[0])
	}
	if sqrResults[1] == nil || sqrResults[1].Cmp(big.NewInt(400)) != 0 {
		t.Errorf("sqrResults[1] = %v, want 400", sqrResults[1])
	}

	// Verify multiplication results
	if mulResults[0] == nil || mulResults[0].Cmp(big.NewInt(12)) != 0 {
		t.Errorf("mulResults[0] = %v, want 12", mulResults[0])
	}
	if mulResults[1] == nil || mulResults[1].Cmp(big.NewInt(30)) != 0 {
		t.Errorf("mulResults[1] = %v, want 30", mulResults[1])
	}
}
