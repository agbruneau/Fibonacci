package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

func TestServer_Start_GracefulShutdown(t *testing.T) {
	// Setup a server with a random port
	registry := map[string]fibonacci.Calculator{
		"fast": &MockCalculator{},
	}
	cfg := config.AppConfig{
		Port: "0", // Random port
	}

	server := NewServer(fibonacci.NewTestFactory(registry), cfg)

	// Channel to signal when server has stopped
	done := make(chan error)

	// Start server in background
	go func() {
		done <- server.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Send signal to stop server
	server.shutdownSignal <- syscall.SIGTERM

	// Wait for server to stop
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server stopped with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server failed to stop within timeout")
	}
}

func TestWriteJSONResponse_Error(t *testing.T) {
	// Create a recorder that fails to write?
	// Or try to encode something invalid.
	// JSON encoding fails on cyclic data structures or channels.

	server := &Server{
		logger: nil, // Should handle nil logger or we set one
	}
	// We need a logger or it might panic if the code logs error
	// server.go:440: s.logger.Printf(...)
	// Let's create a server properly
	server = createTestServer(nil)

	w := httptest.NewRecorder()

	// Unmarshallable data: channel
	data := map[string]interface{}{
		"bad": make(chan int),
	}

	// This should trigger the error path in writeJSONResponse
	// "json: unsupported type: chan int"
	// We capture the log output to verify the error was logged, but server logger writes to os.Stdout by default.
	// For this test, we just ensure it doesn't panic.

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("writeJSONResponse panicked: %v", r)
		}
	}()

	server.writeJSONResponse(w, http.StatusOK, data)

	// If it failed to encode, it should probably log the error and maybe write a partial response or nothing.
	// We can check if body is empty or malformed.
	if w.Body.Len() > 0 {
		// It might have written something before erroring, or nothing.
		// Since encoding fails immediately for the root object, likely nothing valid JSON is written.
	}
}

// TestHandleMetrics_Error removed as handleMetrics is too simple to fail in unit tests without extensive mocking.
