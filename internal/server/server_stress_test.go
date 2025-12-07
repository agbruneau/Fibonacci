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

// ─────────────────────────────────────────────────────────────────────────────
// Stress Test Configuration
// ─────────────────────────────────────────────────────────────────────────────

// stressTestConfig holds configuration for stress tests.
type stressTestConfig struct {
	Concurrency       int           // Number of concurrent goroutines
	RequestsPerClient int           // Requests each goroutine makes
	Timeout           time.Duration // Per-request timeout
	MaxN              uint64        // Maximum N value to request
	DelayBetweenReqs  time.Duration // Delay between requests per client
}

// defaultStressConfig returns the default stress test configuration.
func defaultStressConfig() stressTestConfig {
	return stressTestConfig{
		Concurrency:       100,
		RequestsPerClient: 50,
		Timeout:           30 * time.Second,
		MaxN:              10000,
		DelayBetweenReqs:  0,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Stress Test Helpers
// ─────────────────────────────────────────────────────────────────────────────

// stressTestResult holds the results of a stress test.
type stressTestResult struct {
	TotalRequests    int64
	SuccessCount     int64
	ErrorCount       int64
	RateLimitedCount int64
	Duration         time.Duration
	Errors           []string
}

// RequestsPerSecond calculates the RPS from the result.
func (r *stressTestResult) RequestsPerSecond() float64 {
	if r.Duration.Seconds() == 0 {
		return 0
	}
	return float64(r.TotalRequests) / r.Duration.Seconds()
}

// SuccessRate calculates the success rate as a percentage.
func (r *stressTestResult) SuccessRate() float64 {
	if r.TotalRequests == 0 {
		return 0
	}
	return float64(r.SuccessCount) / float64(r.TotalRequests) * 100
}

// fastMockCalculator is a fast mock calculator for stress testing.
type fastMockCalculator struct{}

func (m *fastMockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	return big.NewInt(int64(n)), nil
}

func (m *fastMockCalculator) Name() string {
	return "Fast Mock Calculator"
}

// slowMockCalculator simulates a slow calculation.
type slowMockCalculator struct {
	delay time.Duration
}

func (m *slowMockCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	select {
	case <-time.After(m.delay):
		return big.NewInt(int64(n)), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *slowMockCalculator) Name() string {
	return "Slow Mock Calculator"
}

// setupStressTestServer creates a test server for stress testing.
func setupStressTestServer(t *testing.T, calc fibonacci.Calculator, rateLimit int) (*httptest.Server, func()) {
	t.Helper()

	registry := map[string]fibonacci.Calculator{
		"fast": calc,
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: rateLimit,
		CleanupInterval:   time.Minute,
	})

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)

	cleanup := func() {
		ts.Close()
		rl.Stop()
	}

	return ts, cleanup
}

// runStressTest executes a stress test with the given configuration.
func runStressTest(t *testing.T, ts *httptest.Server, cfg stressTestConfig) stressTestResult {
	t.Helper()

	var (
		successCount     int64
		errorCount       int64
		rateLimitedCount int64
		wg               sync.WaitGroup
		errorsMu         sync.Mutex
		errors           []string
	)

	start := time.Now()

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client := &http.Client{Timeout: cfg.Timeout}

			for j := 0; j < cfg.RequestsPerClient; j++ {
				n := uint64((clientID*cfg.RequestsPerClient + j) % int(cfg.MaxN))
				url := fmt.Sprintf("%s/calculate?n=%d&algo=fast", ts.URL, n)

				resp, err := client.Get(url)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					errorsMu.Lock()
					if len(errors) < 10 { // Limit stored errors
						errors = append(errors, err.Error())
					}
					errorsMu.Unlock()
					continue
				}

				switch resp.StatusCode {
				case http.StatusOK:
					var result Response
					if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Error == "" {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case http.StatusTooManyRequests:
					atomic.AddInt64(&rateLimitedCount, 1)
				default:
					atomic.AddInt64(&errorCount, 1)
				}

				resp.Body.Close()

				if cfg.DelayBetweenReqs > 0 {
					time.Sleep(cfg.DelayBetweenReqs)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := int64(cfg.Concurrency * cfg.RequestsPerClient)

	return stressTestResult{
		TotalRequests:    totalRequests,
		SuccessCount:     successCount,
		ErrorCount:       errorCount,
		RateLimitedCount: rateLimitedCount,
		Duration:         duration,
		Errors:           errors,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Stress Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestServerUnderLoad performs a comprehensive load test on the server.
// It verifies that the server can handle high concurrent load without
// excessive errors.
func TestServerUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ts, cleanup := setupStressTestServer(t, &fastMockCalculator{}, 100000) // High rate limit
	defer cleanup()

	cfg := defaultStressConfig()
	result := runStressTest(t, ts, cfg)

	// Log results
	t.Logf("Stress Test Results:")
	t.Logf("  Total requests: %d", result.TotalRequests)
	t.Logf("  Successful: %d (%.2f%%)", result.SuccessCount, result.SuccessRate())
	t.Logf("  Errors: %d", result.ErrorCount)
	t.Logf("  Rate limited: %d", result.RateLimitedCount)
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Requests/sec: %.2f", result.RequestsPerSecond())

	// Log first few errors for debugging
	for i, err := range result.Errors {
		t.Logf("  Error %d: %s", i+1, err)
	}

	// Verify error rate is acceptable (less than 1%)
	errorRate := float64(result.ErrorCount) / float64(result.TotalRequests) * 100
	if errorRate > 1.0 {
		t.Errorf("Error rate too high: %.2f%% (expected < 1%%)", errorRate)
	}
}

// TestServerUnderSustainedLoad tests the server under sustained load over time.
func TestServerUnderSustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	ts, cleanup := setupStressTestServer(t, &fastMockCalculator{}, 100000)
	defer cleanup()

	// Run multiple waves of load
	waves := 3
	var totalSuccess, totalErrors int64

	for wave := 0; wave < waves; wave++ {
		cfg := stressTestConfig{
			Concurrency:       50,
			RequestsPerClient: 20,
			Timeout:           10 * time.Second,
			MaxN:              5000,
		}

		result := runStressTest(t, ts, cfg)
		totalSuccess += result.SuccessCount
		totalErrors += result.ErrorCount

		t.Logf("Wave %d: %d success, %d errors, %.2f req/s",
			wave+1, result.SuccessCount, result.ErrorCount, result.RequestsPerSecond())

		// Brief pause between waves
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("Total across %d waves: %d success, %d errors", waves, totalSuccess, totalErrors)

	if totalErrors > (totalSuccess+totalErrors)/100 {
		t.Errorf("Too many errors across waves: %d/%d", totalErrors, totalSuccess+totalErrors)
	}
}

// TestServerWithSlowCalculations tests behavior when calculations are slow.
func TestServerWithSlowCalculations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow calculation test in short mode")
	}

	slowCalc := &slowMockCalculator{delay: 100 * time.Millisecond}
	ts, cleanup := setupStressTestServer(t, slowCalc, 100000)
	defer cleanup()

	cfg := stressTestConfig{
		Concurrency:       20,
		RequestsPerClient: 5,
		Timeout:           5 * time.Second,
		MaxN:              100,
	}

	result := runStressTest(t, ts, cfg)

	t.Logf("Slow calculation test:")
	t.Logf("  Total: %d, Success: %d, Errors: %d", result.TotalRequests, result.SuccessCount, result.ErrorCount)
	t.Logf("  Duration: %v, RPS: %.2f", result.Duration, result.RequestsPerSecond())

	// With slow calculations, we should still have good success rate
	if result.SuccessRate() < 95.0 {
		t.Errorf("Success rate too low with slow calculations: %.2f%%", result.SuccessRate())
	}
}

// TestServerRateLimitingUnderLoad tests that rate limiting works correctly under load.
func TestServerRateLimitingUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limit test in short mode")
	}

	ts, cleanup := setupStressTestServer(t, &fastMockCalculator{}, 10) // Low rate limit
	defer cleanup()

	cfg := stressTestConfig{
		Concurrency:       5,
		RequestsPerClient: 10,
		Timeout:           5 * time.Second,
		MaxN:              100,
	}

	result := runStressTest(t, ts, cfg)

	t.Logf("Rate limiting under load:")
	t.Logf("  Total: %d, Success: %d, Rate Limited: %d", result.TotalRequests, result.SuccessCount, result.RateLimitedCount)

	// We should see rate limiting kick in
	if result.RateLimitedCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}

	// But we should still have some successful requests
	if result.SuccessCount == 0 {
		t.Error("Expected some successful requests even with rate limiting")
	}
}

// TestServerConcurrentEndpoints tests concurrent access to multiple endpoints.
func TestServerConcurrentEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent endpoints test in short mode")
	}

	ts, cleanup := setupStressTestServer(t, &fastMockCalculator{}, 100000)
	defer cleanup()

	endpoints := []string{
		"/health",
		"/algorithms",
		"/calculate?n=100&algo=fast",
		"/calculate?n=1000&algo=fast",
	}

	var wg sync.WaitGroup
	var successCount, errorCount int64

	for i := 0; i < 50; i++ {
		for _, endpoint := range endpoints {
			wg.Add(1)
			go func(ep string) {
				defer wg.Done()

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(ts.URL + ep)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					return
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else if resp.StatusCode != http.StatusTooManyRequests {
					atomic.AddInt64(&errorCount, 1)
				}
			}(endpoint)
		}
	}

	wg.Wait()

	totalRequests := int64(50 * len(endpoints))
	t.Logf("Concurrent endpoints test: %d/%d successful", successCount, totalRequests)

	if errorCount > totalRequests/10 {
		t.Errorf("Too many errors: %d/%d", errorCount, totalRequests)
	}
}

// TestServerGracefulDegradation tests how the server handles partial failures.
func TestServerGracefulDegradation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping graceful degradation test in short mode")
	}

	// Calculator that fails sometimes
	failingCalc := &intermittentFailCalculator{failRate: 0.1}
	ts, cleanup := setupStressTestServer(t, failingCalc, 100000)
	defer cleanup()

	cfg := stressTestConfig{
		Concurrency:       20,
		RequestsPerClient: 10,
		Timeout:           5 * time.Second,
		MaxN:              100,
	}

	result := runStressTest(t, ts, cfg)

	t.Logf("Graceful degradation test:")
	t.Logf("  Total: %d, Success: %d, Errors: %d", result.TotalRequests, result.SuccessCount, result.ErrorCount)

	// With 10% failure rate, we should still have ~90% success
	// Allow some margin for the test
	expectedSuccessRate := 0.80
	if result.SuccessRate() < expectedSuccessRate*100 {
		t.Errorf("Success rate too low: %.2f%% (expected > %.2f%%)",
			result.SuccessRate(), expectedSuccessRate*100)
	}
}

// intermittentFailCalculator is a calculator that fails intermittently for testing.
type intermittentFailCalculator struct {
	failRate float64
	counter  int64
}

func (m *intermittentFailCalculator) Calculate(ctx context.Context, progressChan chan<- fibonacci.ProgressUpdate, calcIndex int, n uint64, opts fibonacci.Options) (*big.Int, error) {
	count := atomic.AddInt64(&m.counter, 1)
	if float64(count%100)/100 < m.failRate {
		return nil, fmt.Errorf("simulated failure")
	}
	return big.NewInt(int64(n)), nil
}

func (m *intermittentFailCalculator) Name() string {
	return "Intermittent Fail Calculator"
}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmark Tests
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkServerConcurrentLoad benchmarks server performance under concurrent load.
func BenchmarkServerConcurrentLoad(b *testing.B) {
	registry := map[string]fibonacci.Calculator{
		"fast": &fastMockCalculator{},
	}
	cfg := config.AppConfig{
		Port:      "0",
		Threshold: 4096,
	}

	rl := NewRateLimiter(RateLimiterConfig{RequestsPerMinute: 1000000})
	defer rl.Stop()

	srv := NewServer(fibonacci.NewTestFactory(registry), cfg, WithRateLimiter(rl))
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url := fmt.Sprintf("%s/calculate?n=%d&algo=fast", ts.URL, i%1000)
			resp, err := client.Get(url)
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
			i++
		}
	})
}
