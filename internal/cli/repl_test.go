package cli

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/fibonacci"
	"github.com/agbru/fibcalc/internal/testutil"
)

func TestNewREPL(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{
		"fast": &fibonacci.MockCalculator{Result: big.NewInt(0)},
	}
	config := REPLConfig{
		DefaultAlgo: "fast",
	}

	repl := NewREPL(registry, config)
	if repl == nil {
		t.Fatal("NewREPL returned nil")
	}
	if repl.currentAlgo != "fast" {
		t.Errorf("Expected default algo 'fast', got '%s'", repl.currentAlgo)
	}
}

func TestNewREPL_DefaultAlgo(t *testing.T) {
	t.Parallel()
	registry := map[string]fibonacci.Calculator{
		"fast": &fibonacci.MockCalculator{Result: big.NewInt(0)},
	}
	config := REPLConfig{
		DefaultAlgo: "", // Empty default
	}

	repl := NewREPL(registry, config)
	if repl.currentAlgo == "" {
		t.Error("Should have picked an available algorithm")
	}
}

func TestProcessCommand(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"mock": &fibonacci.MockCalculator{Result: big.NewInt(10)},
	}
	config := REPLConfig{
		DefaultAlgo: "mock",
		Timeout:     time.Second,
	}

	repl := NewREPL(registry, config)
	var out bytes.Buffer
	repl.SetOutput(&out)

	// Strip colors for testing
	strip := testutil.StripAnsiCodes

	t.Run("calc", func(t *testing.T) {
		repl.processCommand("calc 10")
		// The mock returns result 10. Check if output contains "F(10) =" and "10"
		output := strip(out.String())
		if !strings.Contains(output, "F(10) = 10") {
			t.Errorf("Expected calculation output 'F(10) = 10', got %s", output)
		}
		out.Reset()
	})

	t.Run("calc shorthand", func(t *testing.T) {
		// Mock calculator returns 10 always with current mock setup,
		// but the test expects 5 if we pass 5.
		// We need to update the mock or expect 10.
		// The mock in testing.go is fixed to return Result/Err unless Fn is set.
		// Let's rely on the mock returning 10 for any input,
		// OR we can make a dynamic mock locally or via Fn.
		// Let's update registry for this run or just check "F(5) = 10".
		// Actually, let's use the Fn feature of MockCalculator.

		// Re-initialize with dynamic mock
		dynamicMock := &fibonacci.MockCalculator{
			Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
				return big.NewInt(int64(n)), nil
			},
		}
		repl.registry = map[string]fibonacci.Calculator{"mock": dynamicMock}

		repl.processCommand("c 5")
		output := strip(out.String())
		if !strings.Contains(output, "F(5) = 5") {
			t.Errorf("Expected calculation output 'F(5) = 5', got %s", output)
		}
		out.Reset()
	})

	t.Run("algo", func(t *testing.T) {
		repl.processCommand("algo mock")
		if !strings.Contains(out.String(), "Algorithm changed to") {
			t.Error("Expected algorithm change message")
		}
		out.Reset()
	})

	t.Run("list", func(t *testing.T) {
		repl.processCommand("list")
		if !strings.Contains(out.String(), "Available algorithms") {
			t.Error("Expected list output")
		}
		out.Reset()
	})

	t.Run("status", func(t *testing.T) {
		repl.processCommand("status")
		if !strings.Contains(out.String(), "Current configuration") {
			t.Error("Expected status output")
		}
		out.Reset()
	})

	t.Run("hex", func(t *testing.T) {
		repl.config.HexOutput = false // Ensure starts false
		repl.processCommand("hex")
		if !strings.Contains(out.String(), "Hexadecimal display:") {
			t.Error("Expected hex status message")
		}
		if !repl.config.HexOutput {
			t.Error("HexOutput should be true")
		}
		out.Reset()
	})

	t.Run("compare", func(t *testing.T) {
		repl.processCommand("compare 10")
		if !strings.Contains(out.String(), "Comparison for F(10)") {
			t.Error("Expected comparison output")
		}
		out.Reset()
	})

	t.Run("help", func(t *testing.T) {
		repl.processCommand("help")
		if !strings.Contains(out.String(), "Available commands") {
			t.Error("Expected help output")
		}
		out.Reset()
	})

	t.Run("unknown", func(t *testing.T) {
		repl.processCommand("unknown")
		if !strings.Contains(out.String(), "Unknown command") {
			t.Error("Expected unknown command message")
		}
		out.Reset()
	})

	t.Run("numeric input", func(t *testing.T) {
		// Reset hex output mode which was toggled in previous test
		repl.config.HexOutput = false
		repl.processCommand("20")
		output := strip(out.String())
		if !strings.Contains(output, "F(20) = 20") {
			t.Errorf("Expected numeric input execution 'F(20) = 20', got %s", output)
		}
		out.Reset()
	})

	t.Run("exit", func(t *testing.T) {
		if repl.processCommand("exit") {
			t.Error("Expected exit command to return false")
		}
	})
}

func TestREPLStart(t *testing.T) {
	mock := &fibonacci.MockCalculator{
		Fn: func(ctx context.Context, n uint64) (*big.Int, error) {
			return big.NewInt(int64(n)), nil
		},
	}
	registry := map[string]fibonacci.Calculator{
		"mock": mock,
	}
	config := REPLConfig{DefaultAlgo: "mock"}
	repl := NewREPL(registry, config)

	// Simulate user input
	input := "calc 5\nexit\n"
	repl.SetInput(strings.NewReader(input))
	var out bytes.Buffer
	repl.SetOutput(&out)

	repl.Start()

	output := testutil.StripAnsiCodes(out.String())
	if !strings.Contains(output, "F(5) = 5") {
		t.Errorf("Expected calculation output, got %s", output)
	}
	if !strings.Contains(output, "Goodbye!") {
		t.Error("Expected goodbye message")
	}
}
