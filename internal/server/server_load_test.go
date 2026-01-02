package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agbru/fibcalc/internal/config"
	"github.com/agbru/fibcalc/internal/fibonacci"
)

// mockCalculator is a simple calculator for testing that returns quickly.
type mockCalculator struct {
	delay time.Duration
}

func (m *mockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return big.NewInt(int64(n)), nil
}

func (m *mockCalculator) Name() string {
	return "Mock Calculator"
}

// TestServerConcurrentRequests tests that the server can handle multiple concurrent requests.
func TestServerConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{delay: 10 * time.Millisecond},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	// Disable rate limiting for this test
	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 10000})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	const (
		numRequests   = 100
		numGoroutines = 10
	)

	var (
		successCount int64
		errorCount   int64
		wg           sync.WaitGroup
	)

	requestsPerGoroutine := numRequests / numGoroutines
	wg.Add(numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			client := &http.Client{Timeout: 30 * time.Second}

			for j := 0; j < requestsPerGoroutine; j++ {
				n := (workerID * requestsPerGoroutine) + j + 1
				url := fmt.Sprintf("%s/calculate?n=%d&algo=fast", ts.URL, n)

				resp, err := client.Get(url)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				var result Response
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					atomic.AddInt64(&errorCount, 1)
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK && result.Error == "" {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Load test completed in %v", duration)
	t.Logf("Total requests: %d", numRequests)
	t.Logf("Successful: %d, Errors: %d", successCount, errorCount)
	t.Logf("Requests per second: %.2f", float64(numRequests)/duration.Seconds())

	if errorCount > int64(numRequests/10) {
		t.Errorf("Too many errors: %d out of %d requests", errorCount, numRequests)
	}
}

// TestServerRateLimiting tests that rate limiting works correctly.
func TestServerRateLimiting(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	// Set low rate limit for testing
	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 5})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	var rateLimitedCount int
	for i := 0; i < 10; i++ {
		resp, err := client.Get(ts.URL + "/calculate?n=10&algo=fast")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	if rateLimitedCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}

	t.Logf("Rate limited %d out of 10 requests", rateLimitedCount)
}

// TestServerSecurityHeaders tests that security headers are set correctly.
func TestServerSecurityHeaders(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 100})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-Xss-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		actual := resp.Header.Get(header)
		if actual != expected {
			t.Errorf("Header %s: expected %q, got %q", header, expected, actual)
		}
	}
}

// TestServerMaxNValidation tests that the maximum N value is enforced.
func TestServerMaxNValidation(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	secConfig := DefaultSecurityConfig()
	secConfig.MaxNValue = 1000 // Set low limit for testing

	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 100})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl), WithSecurityConfig(secConfig))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	// Request with N > MaxNValue should fail
	resp, err := http.Get(ts.URL + "/calculate?n=5000&algo=fast")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Message == "" {
		t.Error("Expected error message about maximum N value")
	}
}

// TestServerMetricsEndpoint tests that the /metrics endpoint works correctly.
func TestServerMetricsEndpoint(t *testing.T) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg)
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	// Make a calculation request first
	resp, err := http.Get(ts.URL + "/calculate?n=10&algo=fast")
	if err != nil {
		t.Fatalf("Calculation request failed: %v", err)
	}
	resp.Body.Close()

	// Now check metrics
	resp, err = http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("Metrics request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	// Allow for extra parameters in content type
	if contentType == "" {
		t.Error("Content-Type header is missing")
	}
}

// BenchmarkServerCalculate benchmarks the calculate endpoint.
func BenchmarkServerCalculate(b *testing.B) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 100000})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(ts.URL + "/calculate?n=100&algo=fast")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkServerHealth benchmarks the health endpoint.
func BenchmarkServerHealth(b *testing.B) {
	registry := map[string]fibonacci.Calculator{
		"fast": &mockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 100000})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(ts.URL + "/health")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}
