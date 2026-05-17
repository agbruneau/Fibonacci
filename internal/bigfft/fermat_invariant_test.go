package bigfft

import (
	"strings"
	"testing"
)

// TestInternalPostconditionNotMasked is the [A-06] regression test.
//
// fermat.Mul/Sqr panic on two distinct classes:
//   - CALLER-ARGUMENT mismatch (len(x) != len(y)): bad input — may be
//     surfaced as an error by the boundary recover().
//   - INTERNAL POST-CONDITION ("len(z) > 2n+1", "unexpected carry after
//     normalization"): a bug inside the FFT code itself — must NOT be
//     masked into an opaque error (audit R3.8 intent / A-06), otherwise a
//     genuine regression is silently downgraded to a returned error.
//
// The boundary recover() in fft.go / context.go must re-panic when it
// recovers a fermatInvariant sentinel and only wrap non-sentinel panics
// as errors. recoverFFTBoundary centralizes that decision; this test
// drives it directly.
func TestInternalPostconditionNotMasked(t *testing.T) {
	t.Parallel()

	// Simulate the boundary: a function that recovers like Mul/Sqr do.
	boundary := func(panicVal any) (err error) {
		defer recoverFFTBoundary("test", &err)
		panic(panicVal)
	}

	t.Run("internal invariant re-panics (not masked)", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("[A-06] internal post-condition was masked into an " +
					"error instead of propagating as a panic")
			}
			fi, ok := r.(fermatInvariant)
			if !ok {
				t.Fatalf("[A-06] expected fermatInvariant to propagate, got %T: %v", r, r)
			}
			if !strings.Contains(fi.msg, "len(z) > 2n+1") {
				t.Fatalf("[A-06] unexpected invariant message: %q", fi.msg)
			}
		}()
		// Should re-panic, so err is never observed.
		_ = boundary(fermatInvariant{msg: "len(z) > 2n+1"})
		t.Fatal("[A-06] boundary swallowed an internal invariant panic")
	})

	t.Run("non-sentinel panic still wraps as error", func(t *testing.T) {
		t.Parallel()
		err := boundary("some unexpected runtime panic")
		if err == nil {
			t.Fatal("[A-06] non-sentinel panic should be wrapped as an error")
		}
		if !strings.Contains(err.Error(), "test") {
			t.Fatalf("[A-06] wrapped error lost boundary context: %v", err)
		}
	})
}

// TestInputMismatchStillReturnsError verifies that a genuine caller-
// argument size mismatch (external/bad-input class) is still reported as
// an error through the public boundary, NOT a panic — preserving the
// R3.8 *Safe contract and the recover() fallback for Mul/Sqr.
func TestInputMismatchStillReturnsError(t *testing.T) {
	t.Parallel()

	// MulSafe is the documented error-returning path for argument
	// mismatch; it must keep returning an error (never panic, never
	// sentinel).
	z := make(fermat, 5)
	x := make(fermat, 4)
	y := make(fermat, 6) // len(x) != len(y): caller mismatch
	got, err := z.MulSafe(x, y)
	if err == nil {
		t.Fatal("[A-06] MulSafe must return an error on operand size mismatch")
	}
	if got != nil {
		t.Fatalf("[A-06] MulSafe returned non-nil result on mismatch: %v", got)
	}
	if !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("[A-06] unexpected mismatch error: %v", err)
	}
}
