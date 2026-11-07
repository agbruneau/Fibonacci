package fibonacci

import (
	"context"
	"runtime"
	"testing"
)

// TestExecuteMatrixLoop_ParallelGateMatchesGOMAXPROCS guards FIB-08: the task
// semaphore (getTaskSemaphore) sizes itself from runtime.GOMAXPROCS(0), so the
// matrix parallel gate must too. A gate on runtime.NumCPU() would, on a
// multi-core machine running with GOMAXPROCS=1, still allow parallel
// squaring/multiplication (NumCPU() > 1) even though the runtime is
// restricted to a single OS thread, and any goroutines spawned only
// serialize behind the size-1 semaphore. Both must agree on GOMAXPROCS(0).
func TestExecuteMatrixLoop_ParallelGateMatchesGOMAXPROCS(t *testing.T) {
	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	framework := NewMatrixFramework()
	var sawParallel bool
	framework.SquareFunc = func(ctx context.Context, dest, mat *matrix, state *matrixState, inParallel bool, fftThreshold int) error {
		if inParallel {
			sawParallel = true
		}
		return squareSymmetricMatrix(ctx, dest, mat, state, inParallel, fftThreshold)
	}

	state := acquireMatrixState()
	defer releaseMatrixState(state)

	opts := Options{ParallelThreshold: 1} // lowest possible threshold: any nonzero matrix bit length qualifies

	_, err := framework.ExecuteMatrixLoop(context.Background(), noopReporter, 100, opts, state)
	if err != nil {
		t.Fatalf("ExecuteMatrixLoop() error = %v", err)
	}

	if sawParallel {
		t.Errorf("SquareFunc invoked with inParallel=true while GOMAXPROCS=1; gate must use runtime.GOMAXPROCS(0), not runtime.NumCPU()")
	}
}

// TestGetTaskSemaphore_SizedByGOMAXPROCS locks in that the shared task
// semaphore capacity tracks runtime.GOMAXPROCS(0) — the same source the
// matrix parallel gate now reads — so the two never disagree again (FIB-08).
func TestGetTaskSemaphore_SizedByGOMAXPROCS(t *testing.T) {
	sem := getTaskSemaphore()
	if got, want := cap(sem), runtime.GOMAXPROCS(0); got != want {
		t.Errorf("cap(getTaskSemaphore()) = %d, want runtime.GOMAXPROCS(0) = %d", got, want)
	}
}
