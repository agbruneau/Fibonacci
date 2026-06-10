package fibonacci

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// TestMain aligns the global zerolog level with production (app.New sets
// InfoLevel). Without it the global logger defaults to trace and
// CalculateWithObservers emits one JSON line per call, interleaving log
// output with `go test -bench` result lines and breaking benchstat parsing.
func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	os.Exit(m.Run())
}
