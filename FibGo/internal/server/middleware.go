// Package server provides the HTTP server implementation for the Fibonacci calculator API.
package server

import (
	"net/http"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Server Options for Middleware Integration
// ─────────────────────────────────────────────────────────────────────────────

// WithRateLimiter sets a custom rate limiter for the server.
//
// Parameters:
//   - rl: The rate limiter to use.
//
// Returns:
//   - Option: A functional option that configures the server's rate limiter.
func WithRateLimiter(rl *RateLimiter) Option {
	return func(s *Server) {
		s.rateLimiter = rl
	}
}

// WithSecurityConfig sets a custom security configuration for the server.
//
// Parameters:
//   - config: The security configuration.
//
// Returns:
//   - Option: A functional option that configures the server's security settings.
func WithSecurityConfig(config SecurityConfig) Option {
	return func(s *Server) {
		s.securityConfig = config
	}
}

// WithMaxN sets the maximum allowed value for the 'n' parameter.
// This helps prevent DoS attacks via extremely large calculations.
//
// Parameters:
//   - maxN: The maximum allowed value.
//
// Returns:
//   - Option: A functional option that configures the maximum N value.
func WithMaxN(maxN uint64) Option {
	return func(s *Server) {
		s.securityConfig.MaxNValue = maxN
	}
}

// loggingMiddleware wraps an http.HandlerFunc to log the details of each request.
// It records the HTTP method, URL path, remote address, and the duration required
// to process the request.
//
// Parameters:
//   - next: The next handler in the chain.
//
// Returns:
//   - http.HandlerFunc: A new handler with logging capability.
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next(w, r)

		duration := time.Since(start)
		s.logger.Printf("%s %s completed in %v", r.Method, r.URL.Path, duration)
	}
}
