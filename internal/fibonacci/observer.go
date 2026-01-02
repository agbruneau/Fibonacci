// Package fibonacci provides implementations for calculating Fibonacci numbers.
// This file contains the Observer pattern implementation for progress reporting.
package fibonacci

import (
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// Observer Pattern Interfaces
// ─────────────────────────────────────────────────────────────────────────────

// ProgressObserver defines the interface for observing progress events.
// Implementations receive notifications when calculation progress changes,
// enabling decoupled handling of progress updates for UI, logging, metrics, etc.
type ProgressObserver interface {
	// Update is called when progress changes.
	//
	// Parameters:
	//   - calcIndex: The calculator instance identifier (for concurrent calculations)
	//   - progress: The normalized progress value (0.0 to 1.0)
	Update(calcIndex int, progress float64)
}

// ─────────────────────────────────────────────────────────────────────────────
// Progress Subject (Observable)
// ─────────────────────────────────────────────────────────────────────────────

// ProgressSubject manages observer registration and notification for progress events.
// It implements the Subject part of the Observer pattern, allowing multiple observers
// to be notified of progress updates without tight coupling between the calculator
// and its consumers.
//
// ProgressSubject is safe for concurrent use.
type ProgressSubject struct {
	observers []ProgressObserver
	mu        sync.RWMutex
}

// NewProgressSubject creates a new subject for managing progress observers.
//
// Returns:
//   - *ProgressSubject: A new, empty subject ready to accept observers.
func NewProgressSubject() *ProgressSubject {
	return &ProgressSubject{
		observers: make([]ProgressObserver, 0),
	}
}

// Register adds an observer to receive progress updates.
// Observers are notified in the order they are registered.
//
// Parameters:
//   - observer: The observer to add. If nil, this call is a no-op.
func (s *ProgressSubject) Register(observer ProgressObserver) {
	if observer == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, observer)
}

// Unregister removes an observer from receiving updates.
// If the observer is not found, this call is a no-op.
//
// Parameters:
//   - observer: The observer to remove.
func (s *ProgressSubject) Unregister(observer ProgressObserver) {
	if observer == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, o := range s.observers {
		if o == observer {
			// Remove observer while preserving order
			s.observers = append(s.observers[:i], s.observers[i+1:]...)
			return
		}
	}
}

// Notify sends a progress update to all registered observers.
// Observers are notified synchronously in registration order.
//
// Parameters:
//   - calcIndex: The calculator instance identifier.
//   - progress: The normalized progress value (0.0 to 1.0).
func (s *ProgressSubject) Notify(calcIndex int, progress float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, observer := range s.observers {
		observer.Update(calcIndex, progress)
	}
}

// ObserverCount returns the number of registered observers.
// This is primarily useful for testing and diagnostics.
//
// Returns:
//   - int: The number of registered observers.
func (s *ProgressSubject) ObserverCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.observers)
}

// AsProgressReporter returns a ProgressReporter function that notifies all observers.
// This provides backward compatibility with existing calculator implementations that
// use the functional ProgressReporter type.
//
// Parameters:
//   - calcIndex: The calculator instance identifier to include in notifications.
//
// Returns:
//   - ProgressReporter: A function that can be passed to core calculators.
func (s *ProgressSubject) AsProgressReporter(calcIndex int) ProgressReporter {
	return func(progress float64) {
		s.Notify(calcIndex, progress)
	}
}
