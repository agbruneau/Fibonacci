package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiting algorithm.
// It limits the number of requests per client (identified by IP) within a time window.
type RateLimiter struct {
	mu       sync.Mutex // Optimized: Mutex is faster than RWMutex for write-heavy workloads
	clients  map[string]*clientLimiter
	rate     int           // Maximum requests per window
	window   time.Duration // Time window duration
	cleanup  time.Duration // Cleanup interval for expired entries
	stopChan chan struct{}
}

// clientLimiter tracks the request count and window start time for a single client.
type clientLimiter struct {
	tokens      int
	windowStart time.Time
}

// RateLimiterConfig holds configuration for the rate limiter.
type RateLimiterConfig struct {
	// RequestsPerMinute is the maximum number of requests allowed per minute per client.
	// Default: 60
	RequestsPerMinute int
	// CleanupInterval is how often to clean up expired client entries.
	// Default: 5 minutes
	CleanupInterval time.Duration
}

// DefaultRateLimiterConfig returns the default rate limiter configuration.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerMinute: 60,
		CleanupInterval:   5 * time.Minute,
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration.
//
// Parameters:
//   - config: The rate limiter configuration.
//
// Returns:
//   - *RateLimiter: A new rate limiter instance.
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	if config.RequestsPerMinute <= 0 {
		config.RequestsPerMinute = 60
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	rl := &RateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     config.RequestsPerMinute,
		window:   time.Minute,
		cleanup:  config.CleanupInterval,
		stopChan: make(chan struct{}),
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request from the given client should be allowed.
//
// Parameters:
//   - clientIP: The client's IP address.
//
// Returns:
//   - bool: true if the request is allowed, false if rate limited.
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientIP]

	if !exists {
		rl.clients[clientIP] = &clientLimiter{
			tokens:      rl.rate - 1,
			windowStart: now,
		}
		return true
	}

	// Check if we need to reset the window
	if now.Sub(client.windowStart) >= rl.window {
		client.tokens = rl.rate - 1
		client.windowStart = now
		return true
	}

	// Check if we have tokens available
	if client.tokens > 0 {
		client.tokens--
		return true
	}

	return false
}

// cleanupLoop periodically removes expired client entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, client := range rl.clients {
				if now.Sub(client.windowStart) > rl.window*2 {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop stops the rate limiter's background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// RateLimitMiddleware wraps an http.HandlerFunc with rate limiting.
//
// Parameters:
//   - rl: The rate limiter to use.
//   - next: The next handler in the chain.
//
// Returns:
//   - http.HandlerFunc: A new handler with rate limiting capability.
func RateLimitMiddleware(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if !rl.Allow(clientIP) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Too Many Requests","message":"Rate limit exceeded. Please try again later."}`))
			return
		}

		next(w, r)
	}
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For and X-Real-IP headers for proxied requests.
//
// The function follows this priority:
//  1. X-Forwarded-For header (first IP in the comma-separated list)
//  2. X-Real-IP header
//  3. RemoteAddr (with port stripped)
//
// Parameters:
//   - r: The HTTP request.
//
// Returns:
//   - string: The client IP address.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list (client's original IP)
		return extractFirstIP(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (strip port if present)
	return stripPort(r.RemoteAddr)
}

// extractFirstIP extracts the first IP address from a comma-separated list.
// This is typically used for X-Forwarded-For headers where the first IP
// represents the original client.
//
// Parameters:
//   - xff: A comma-separated list of IP addresses.
//
// Returns:
//   - string: The first IP address, trimmed of whitespace.
func extractFirstIP(xff string) string {
	if idx := strings.IndexByte(xff, ','); idx != -1 {
		return strings.TrimSpace(xff[:idx])
	}
	return strings.TrimSpace(xff)
}

// stripPort removes the port from an address string.
// It uses net.SplitHostPort for proper handling of both IPv4 and IPv6 addresses.
//
// Examples:
//   - "127.0.0.1:8080" -> "127.0.0.1"
//   - "[::1]:8080" -> "::1"
//   - "192.168.1.1" -> "192.168.1.1" (no port)
//
// Parameters:
//   - addr: The address string, potentially with a port.
//
// Returns:
//   - string: The IP address without the port.
func stripPort(addr string) string {
	// Use net.SplitHostPort for proper IPv4/IPv6 handling
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If parsing fails, the address might not have a port
		// Return as-is after removing any brackets from IPv6
		return strings.Trim(addr, "[]")
	}
	return host
}
