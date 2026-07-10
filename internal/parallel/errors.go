package parallel

import "sync"

// ErrorCollector collects the first error from parallel goroutines.
// SetError is safe for concurrent use; Err() and reset() are not — read the
// result only after every SetError-ing goroutine has finished (see Err's doc
// and the Usage pattern below).
//
// Usage:
//
//	var ec parallel.ErrorCollector
//	var wg sync.WaitGroup
//	wg.Add(2)
//	go func() {
//	    defer wg.Done()
//	    ec.SetError(doWork1())
//	}()
//	go func() {
//	    defer wg.Done()
//	    ec.SetError(doWork2())
//	}()
//	wg.Wait()
//	if err := ec.Err(); err != nil {
//	    return err
//	}
type ErrorCollector struct {
	once sync.Once
	err  error
}

// SetError records an error if one hasn't been recorded yet.
// Nil errors are ignored. This method is thread-safe.
//
// Parameters:
//   - err: The error to record (nil is ignored).
func (c *ErrorCollector) SetError(err error) {
	if err != nil {
		c.once.Do(func() {
			c.err = err
		})
	}
}

// Err returns the first recorded error, or nil if no error was recorded.
// This method reads c.err with no synchronization of its own: it is safe
// only when called after every SetError-ing goroutine has finished (e.g.
// after wg.Wait()), which establishes the happens-before edge. All current
// callers follow this pattern. Calling Err() concurrently with a live
// SetError is a data race.
//
// Returns:
//   - error: The first recorded error or nil.
func (c *ErrorCollector) Err() error {
	return c.err
}

// reset resets the collector for reuse.
// WARNING: This is NOT thread-safe and should only be called when
// no goroutines are using the collector.
func (c *ErrorCollector) reset() {
	c.once = sync.Once{}
	c.err = nil
}
