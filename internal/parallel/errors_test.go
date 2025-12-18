package parallel

import (
	"errors"
	"sync"
	"testing"
)

func TestErrorCollector_SetError(t *testing.T) {
	t.Parallel()
	ec := &ErrorCollector{}
	expectedErr := errors.New("first error")
	otherErr := errors.New("second error")

	// Set the first error
	ec.SetError(expectedErr)
	if ec.Err() != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, ec.Err())
	}

	// Try to set a second error, should be ignored
	ec.SetError(otherErr)
	if ec.Err() != expectedErr {
		t.Errorf("Expected error %v to persist, got %v", expectedErr, ec.Err())
	}

	// Try to set nil, should be ignored
	ec.SetError(nil)
	if ec.Err() != expectedErr {
		t.Errorf("Expected error %v to persist after nil set, got %v", expectedErr, ec.Err())
	}
}

func TestErrorCollector_Concurrency(t *testing.T) {
	t.Parallel()
	ec := &ErrorCollector{}
	var wg sync.WaitGroup
	numGoroutines := 100

	// Create a channel to ensure all goroutines are ready before starting
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ec.SetError(errors.New("error from goroutine"))
		}()
	}

	close(start)
	wg.Wait()

	if ec.Err() == nil {
		t.Error("Expected an error to be collected, got nil")
	}
	if ec.Err().Error() != "error from goroutine" {
		t.Errorf("Expected 'error from goroutine', got %v", ec.Err())
	}
}

func TestErrorCollector_Reset(t *testing.T) {
	t.Parallel()
	ec := &ErrorCollector{}
	err := errors.New("test error")

	ec.SetError(err)
	if ec.Err() != err {
		t.Errorf("Expected error %v, got %v", err, ec.Err())
	}

	ec.Reset()
	if ec.Err() != nil {
		t.Errorf("Expected nil error after reset, got %v", ec.Err())
	}

	// Should be able to set error again after reset
	newErr := errors.New("new error")
	ec.SetError(newErr)
	if ec.Err() != newErr {
		t.Errorf("Expected new error %v, got %v", newErr, ec.Err())
	}
}
