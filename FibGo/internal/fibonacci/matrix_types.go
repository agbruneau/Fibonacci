package fibonacci

import (
	"math/big"
	"sync"
)

// matrix represents a 2x2 matrix of *big.Int values.
// It is a fundamental data structure for the matrix exponentiation algorithm.
// The fields a, b, c, and d correspond to the elements of the matrix:
//
//	[ a b ]
//	[ c d ]
type matrix struct{ a, b, c, d *big.Int }

// newMatrix allocates and returns a new 2x2 matrix.
// It initializes each of its elements with a new *big.Int, which is a
// convenience for ensuring that matrices are correctly instantiated.
//
// Returns:
//   - *matrix: A pointer to the newly created matrix.
func newMatrix() *matrix {
	// We return a matrix struct with nil pointers, as they will be populated from the pool
	return &matrix{
		a: new(big.Int),
		b: new(big.Int),
		c: new(big.Int),
		d: new(big.Int),
	}
}

// Set copies the values from another matrix into the receiver matrix.
// This is a deep copy, ensuring that the underlying *big.Int values are
// duplicated.
//
// Parameters:
//   - other: The source matrix to copy from.
func (m *matrix) Set(other *matrix) {
	m.a.Set(other.a)
	m.b.Set(other.b)
	m.c.Set(other.c)
	m.d.Set(other.d)
}

// SetIdentity configures the matrix as an identity matrix.
// The identity matrix is the multiplicative identity for matrix multiplication,
// and is defined as:
//
//	[ 1 0 ]
//	[ 0 1 ]
func (m *matrix) SetIdentity() {
	m.a.SetInt64(1) // Same as the neutral element of a matrix
	m.b.SetInt64(0)
	m.c.SetInt64(0)
	m.d.SetInt64(1)
}

// SetBaseQ configures the matrix as the base Fibonacci matrix, Q.
// Powers of this matrix are used to generate Fibonacci numbers. It is defined
// as:
//
//	[ 1 1 ]
//	[ 1 0 ]
func (m *matrix) SetBaseQ() {
	m.a.SetInt64(1) // Base Q of the Fibonacci recurrence
	m.b.SetInt64(1)
	m.c.SetInt64(1)
	m.d.SetInt64(0)
}

// matrixState aggregates variables for the matrix exponentiation algorithm.
// The temporary variables (p1-p7, s1-s8) are specifically designed to support
// the memory requirements of Strassen's matrix multiplication algorithm,
// allowing the entire operation to proceed without any memory allocations in the
// hot path.
type matrixState struct {
	res, p, tempMatrix *matrix
	// Temporaries for Strassen's algorithm products
	p1, p2, p3, p4, p5, p6, p7 *big.Int
	// Temporaries for Strassen's algorithm sums/differences
	s1, s2, s3, s4, s5, s6, s7, s8 *big.Int
	// General purpose temporaries for symmetric squaring
	t1, t2, t3, t4, t5 *big.Int
}

// Reset resets the state for a new use.
// It sets the result matrix to identity and the base power matrix to Q.
func (s *matrixState) Reset() {
	s.res.SetIdentity()
	s.p.SetBaseQ()
}

// matrixStatePool is a `sync.Pool` for `matrixState` objects. Object pools are a
// performance optimization technique used to reduce memory allocation and garbage
// collector overhead. By reusing `matrixState` objects, the application can avoid
// the cost of creating and destroying them for each calculation, which is
// particularly beneficial in a high-performance, concurrent context.
var matrixStatePool = sync.Pool{
	New: func() any {
		// Fully initialize the state with all big.Ints
		return &matrixState{
			res:        newMatrix(),
			p:          newMatrix(),
			tempMatrix: newMatrix(),
			p1:         new(big.Int),
			p2:         new(big.Int),
			p3:         new(big.Int),
			p4:         new(big.Int),
			p5:         new(big.Int),
			p6:         new(big.Int),
			p7:         new(big.Int),
			s1:         new(big.Int),
			s2:         new(big.Int),
			s3:         new(big.Int),
			s4:         new(big.Int),
			s5:         new(big.Int),
			s6:         new(big.Int),
			s7:         new(big.Int),
			s8:         new(big.Int),
			t1:         new(big.Int),
			t2:         new(big.Int),
			t3:         new(big.Int),
			t4:         new(big.Int),
			t5:         new(big.Int),
		}
	},
}

// acquireMatrixState gets a state from the pool and resets it.
// The returned state must be released using releaseMatrixState, preferably with defer:
//
//	state := acquireMatrixState()
//	defer releaseMatrixState(state)
//
// This ensures the state is returned to the pool even if an error occurs or a panic is triggered.
//
// Returns:
//   - *matrixState: A fresh or reused matrixState.
func acquireMatrixState() *matrixState {
	s := matrixStatePool.Get().(*matrixState)
	s.Reset()
	return s
}

// releaseMatrixState puts a state back into the pool.
// This should be called with defer immediately after acquireMatrixState to ensure
// proper resource cleanup even in case of errors or panics:
//
//	state := acquireMatrixState()
//	defer releaseMatrixState(state)
//
// Parameters:
//   - s: The matrixState to return to the pool. Safe to call with nil.
func releaseMatrixState(s *matrixState) {
	if matrixStateOversized(s) {
		return
	}
	matrixStatePool.Put(s)
}

// matrixStateOversized returns true when any big.Int held by s — the three
// matrices' elements or any Strassen/squaring temporary — exceeds
// MaxPooledBitLen, so the state is dropped to GC instead of pinning it.
func matrixStateOversized(s *matrixState) bool {
	for _, m := range []*matrix{s.res, s.p, s.tempMatrix} {
		if checkLimit(m.a) || checkLimit(m.b) || checkLimit(m.c) || checkLimit(m.d) {
			return true
		}
	}
	for _, z := range []*big.Int{
		s.p1, s.p2, s.p3, s.p4, s.p5, s.p6, s.p7,
		s.s1, s.s2, s.s3, s.s4, s.s5, s.s6, s.s7, s.s8,
		s.t1, s.t2, s.t3, s.t4, s.t5,
	} {
		if checkLimit(z) {
			return true
		}
	}
	return false
}
