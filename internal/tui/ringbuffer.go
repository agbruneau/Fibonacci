package tui

// Ring is a fixed-capacity circular buffer that overwrites the oldest
// element when full. It is generic over the element type T and is safe for
// single-goroutine use only (the TUI runs everything on the bubbletea
// goroutine, so no synchronisation is needed here).
//
// The buffer is intentionally minimal: Push, Len, Cap, Last, Slice, Resize,
// and Reset cover every call site in the tui package (sparkline samples and
// log entries). Adding methods speculatively would violate the surgical-change
// guideline.
type Ring[T any] struct {
	data  []T
	head  int
	count int
}

// RingBuffer is the float64 instantiation of Ring, kept as an alias so that
// existing call sites (chart.go, sparkline_test.go) can keep using the
// non-generic name. Audit task R4.8 introduced the generic Ring so LogsModel
// can share the same fixed-capacity storage with strings.
type RingBuffer = Ring[float64]

// NewRing creates a generic ring buffer with the given capacity. A capacity
// of 0 or below is silently raised to 1 to keep the indexing arithmetic
// well-defined.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity <= 0 {
		capacity = 1
	}
	return &Ring[T]{data: make([]T, capacity)}
}

// NewRingBuffer creates a float64 ring buffer. Kept as a non-generic
// constructor so existing call sites (sparkline samples, sparkline_test.go,
// chart.go) can keep using NewRingBuffer(n) without specifying a type
// parameter.
func NewRingBuffer(capacity int) *RingBuffer {
	return NewRing[float64](capacity)
}

// Push adds a sample, overwriting the oldest if full.
func (r *Ring[T]) Push(v T) {
	r.data[r.head] = v
	r.head = (r.head + 1) % len(r.data)
	if r.count < len(r.data) {
		r.count++
	}
}

// Len returns the number of valid samples currently stored.
func (r *Ring[T]) Len() int { return r.count }

// Cap returns the buffer capacity.
func (r *Ring[T]) Cap() int { return len(r.data) }

// Last returns the most recent sample, or the zero value of T if empty.
func (r *Ring[T]) Last() T {
	var zero T
	if r.count == 0 {
		return zero
	}
	idx := r.head - 1
	if idx < 0 {
		idx = len(r.data) - 1
	}
	return r.data[idx]
}

// Slice returns samples in chronological order (oldest first). Returns nil
// when the buffer is empty so that callers can treat "no data" as a falsy
// value (the original sparkline contract relied on this).
func (r *Ring[T]) Slice() []T {
	if r.count == 0 {
		return nil
	}
	result := make([]T, r.count)
	start := r.head - r.count
	if start < 0 {
		start += len(r.data)
	}
	for i := range r.count {
		result[i] = r.data[(start+i)%len(r.data)]
	}
	return result
}

// Resize changes the capacity, preserving the most recent samples that fit.
// A resize to the current capacity is a no-op (avoids reallocation churn
// when SetSize is called repeatedly with the same width).
func (r *Ring[T]) Resize(newCap int) {
	if newCap <= 0 {
		newCap = 1
	}
	if newCap == len(r.data) {
		return
	}
	old := r.Slice()
	r.data = make([]T, newCap)
	r.head = 0
	r.count = 0
	start := 0
	if len(old) > newCap {
		start = len(old) - newCap
	}
	for _, v := range old[start:] {
		r.Push(v)
	}
}

// Reset clears all samples without releasing the underlying storage.
func (r *Ring[T]) Reset() {
	r.head = 0
	r.count = 0
}
