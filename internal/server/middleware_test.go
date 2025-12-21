package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Unit tests for middleware functions

func TestExtractFirstIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"127.0.0.1", "127.0.0.1"},
		{"127.0.0.1, 192.168.1.1", "127.0.0.1"},
		{"10.0.0.1, 10.0.0.2, 10.0.0.3", "10.0.0.1"},
		{"", ""},
		{"   1.2.3.4   ", "1.2.3.4"},
	}

	for _, tt := range tests {
		got := extractFirstIP(tt.input)
		if got != tt.expected {
			t.Errorf("extractFirstIP(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"127.0.0.1:8080", "127.0.0.1"},
		{"192.168.1.1", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"[::1]", "::1"},
	}

	for _, tt := range tests {
		got := stripPort(tt.input)
		if got != tt.expected {
			t.Errorf("stripPort(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			remote:   "9.9.9.9:1234",
			expected: "1.2.3.4",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "5.6.7.8"},
			remote:   "9.9.9.9:1234",
			expected: "5.6.7.8",
		},
		{
			name:     "RemoteAddr",
			headers:  map[string]string{},
			remote:   "9.9.9.9:1234",
			expected: "9.9.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remote

			got := getClientIP(req)
			if got != tt.expected {
				t.Errorf("getClientIP() = %q; want %q", got, tt.expected)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		input    []string
		sep      string
		expected string
	}{
		{[]string{}, ",", ""},
		{[]string{"a"}, ",", "a"},
		{[]string{"a", "b"}, ",", "a,b"},
		{[]string{"a", "b", "c"}, " - ", "a - b - c"},
	}

	for _, tt := range tests {
		// Optimization: Replaced custom joinStrings with strings.Join
		got := strings.Join(tt.input, tt.sep)
		if got != tt.expected {
			t.Errorf("strings.Join(%v, %q) = %q; want %q", tt.input, tt.sep, got, tt.expected)
		}
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		CleanupInterval:   10 * time.Millisecond,
	})
	// Override window for test
	rl.window = 10 * time.Millisecond

	rl.Allow("1.2.3.4")

	rl.mu.Lock()
	if len(rl.clients) != 1 {
		t.Error("Should have 1 client")
	}
	rl.mu.Unlock()

	// Wait for cleanup (needs > 2*window = 20ms)
	time.Sleep(50 * time.Millisecond)

	rl.mu.Lock()
	if len(rl.clients) != 0 {
		t.Error("Client should have been cleaned up")
	}
	rl.mu.Unlock()

	rl.Stop()
}
