package server

import (
	"net/http"
	"strings"
)

// SecurityConfig holds configuration for security headers.
type SecurityConfig struct {
	// EnableCORS enables Cross-Origin Resource Sharing headers.
	EnableCORS bool
	// AllowedOrigins specifies allowed CORS origins. Use "*" for all origins.
	AllowedOrigins []string
	// AllowedMethods specifies allowed HTTP methods for CORS.
	AllowedMethods []string
	// MaxNValue is the maximum allowed value for the 'n' parameter.
	// This prevents DoS attacks via extremely large calculations.
	// Default: 1_000_000_000 (1 billion)
	MaxNValue uint64
}

// DefaultSecurityConfig returns the default security configuration.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableCORS:     true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		MaxNValue:      1_000_000_000, // 1 billion - reasonable limit
	}
}

// SecurityMiddleware adds security headers to HTTP responses.
// It implements best practices for API security:
//   - Content Security Policy (CSP)
//   - X-Content-Type-Options
//   - X-Frame-Options
//   - X-XSS-Protection
//   - Referrer-Policy
//   - CORS headers (if enabled)
//
// Parameters:
//   - config: The security configuration.
//   - next: The next handler in the chain.
//
// Returns:
//   - http.HandlerFunc: A new handler with security headers.
func SecurityMiddleware(config SecurityConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// CORS headers
		if config.EnableCORS {
			origin := r.Header.Get("Origin")
			allowedOrigin := ""

			// Check if origin is allowed
			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					allowedOrigin = allowed
					break
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next(w, r)
	}
}
