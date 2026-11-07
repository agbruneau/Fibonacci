package fibonacci

import (
	"sync"
	"testing"
)

// resetStatePoolForTest gives the caller a statePool with no leftovers, and
// restores an equally empty one afterwards so the next test does not inherit
// whatever this one put back.
//
// statePool is package-global mutable state shared by every test in the
// package, which is exactly what the book warns against (ch. 7, "Avoid using
// shared or global state between tests"). Pooled CalculationState values carry
// an arena whose size depends on the n of whichever test allocated it, so a
// test that asserts anything about arena or cache retention is really asserting
// something about the order the suite happened to run in. That is what made
// TestStateBump_PinnedAcrossCachedCalls flaky.
//
// Only for tests that genuinely depend on pool contents. Everything else should
// stay parallel and pool-agnostic.
//
// The caller MUST NOT be parallel: Go resumes parallel tests only after the
// sequential pass finishes, so a non-parallel test is the only one guaranteed
// not to overlap with a test that could refill the pool.
func resetStatePoolForTest(t *testing.T) {
	t.Helper()

	// Capture only the constructor, never the pool itself: copying a sync.Pool
	// is a vet copylocks error.
	newFn := statePool.New

	statePool = sync.Pool{New: newFn}
	t.Cleanup(func() {
		statePool = sync.Pool{New: newFn}
	})
}
