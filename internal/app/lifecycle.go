package app

import (
	"context"
	"os/signal"
	"syscall"
	"time"
)

// SetupContext creates a context with a timeout applied.
// The returned cancel function should be deferred to ensure cleanup.
//
// Parameters:
//   - ctx: The parent context.
//   - timeout: The duration after which the context will be canceled.
//
// Returns:
//   - context.Context: A new context with the timeout applied.
//   - context.CancelFunc: A function to cancel the context (should be deferred).
func SetupContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// SetupSignals creates a context that will be canceled when the application
// receives SIGINT (Ctrl+C) or SIGTERM signals. This enables graceful shutdown.
//
// Parameters:
//   - ctx: The parent context.
//
// Returns:
//   - context.Context: A new context that will be canceled on signal receipt.
//   - context.CancelFunc: A function to stop listening for signals (should be deferred).
func SetupSignals(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}

// SetupLifecycle combines timeout and signal handling into a single call.
// It creates a context that will be canceled either when the timeout expires
// or when a termination signal is received, whichever happens first.
//
// Parameters:
//   - ctx: The parent context.
//   - timeout: The maximum duration for the operation.
//
// Returns:
//   - context.Context: A context with both timeout and signal handling.
//   - CancelFuncs: A struct containing both cancel functions for cleanup.
func SetupLifecycle(ctx context.Context, timeout time.Duration) (context.Context, *CancelFuncs) {
	ctx, cancelTimeout := SetupContext(ctx, timeout)
	ctx, stopSignals := SetupSignals(ctx)

	return ctx, &CancelFuncs{
		CancelTimeout: cancelTimeout,
		StopSignals:   stopSignals,
	}
}

// CancelFuncs holds the cancel functions for lifecycle management.
// Both functions should be called (typically via defer) to ensure proper cleanup.
type CancelFuncs struct {
	// CancelTimeout cancels the timeout context.
	CancelTimeout context.CancelFunc
	// StopSignals stops listening for OS signals.
	StopSignals context.CancelFunc
}

// Cleanup calls both cancel functions to release resources.
// This is a convenience method for use with defer.
func (c *CancelFuncs) Cleanup() {
	if c.StopSignals != nil {
		c.StopSignals()
	}
	if c.CancelTimeout != nil {
		c.CancelTimeout()
	}
}
