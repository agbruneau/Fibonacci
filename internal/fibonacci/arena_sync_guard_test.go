package fibonacci

import (
	"testing"

	"github.com/agbruneau/FibGo/internal/fibonacci/memory"
)

// TestArenaSizingMatchesMemoryArena pins the CLAUDE.md invariant that the
// arena over-sizing multiplier lives in two independent literals —
// acquireSizingForN here and memory.arenaTotalWords — which must stay
// synchronized. Each package's own guardian compares its literal to a local
// naive copy, so a x8/x10 desync keeps both suites green while the state's
// sizing and the arena's actual capacity disagree. This is the only test
// that compares the two functions against each other; memory's literal is
// observed through its exported surface (NewCalculationArena sizes its
// buffer with arenaTotalWords for n >= 1000).
func TestArenaSizingMatchesMemoryArena(t *testing.T) {
	t.Parallel()

	// n >= 1000 (below that NewCalculationArena returns an empty arena) and
	// n <= 10M to keep the largest allocated buffer around 8.7 MB.
	samples := []uint64{1000, 1024, 4097, 65_536, 100_000, 999_999, 1_000_000, 10_000_000}
	for _, n := range samples {
		_, totalWords := acquireSizingForN(n)
		arenaCap := memory.NewCalculationArena(n).CapacityWords()
		if totalWords != arenaCap {
			t.Errorf("arena multiplier drift at n=%d: acquireSizingForN totalWords=%d, memory arena capacity=%d",
				n, totalWords, arenaCap)
		}
	}
}
