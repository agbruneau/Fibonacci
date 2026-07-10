package bigfft

import (
	"math/big"
	"strings"
	"testing"
)

// TestFermatPostConditionPanicClassifier validates that the sentinel list
// covers exactly the panic messages emitted from the four Fermat
// post-condition sites in fermat.go. ADR-0002 / Audit-PRD E2-R1.
func TestFermatPostConditionPanicClassifier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		want bool
	}{
		{"len(z) > 2n+1 sentinel", "len(z) > 2n+1", true},
		{"Mul carry sentinel", "fermat.Mul: unexpected carry after normalization", true},
		{"Sqr carry sentinel", "fermat.Sqr: unexpected carry after normalization", true},
		{"unrelated panic string", "Mul: len(x) != len(y)", false},
		{"non-string panic", struct{ msg string }{"foo"}, false},
		{"nil", nil, false},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isFermatPostConditionPanic(tc.in)
			if got != tc.want {
				t.Errorf("isFermatPostConditionPanic(%v): got %v want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestFermatPanicToError validates the shared panic-policy helper used by
// all eight public entry points (Mul/MulTo/Sqr/SqrTo and the four
// *WithContext variants). Sentinels re-propagate; everything else becomes
// an error naming the entry point. ADR-0002.
func TestFermatPanicToError(t *testing.T) {
	t.Parallel()

	t.Run("non-sentinel panic becomes named error", func(t *testing.T) {
		t.Parallel()
		err := fermatPanicToError("Mul: len(x) != len(y)", "MulWithContext")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if want := "panic in bigfft.MulWithContext"; !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	})

	t.Run("post-condition sentinel re-propagates", func(t *testing.T) {
		t.Parallel()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected sentinel panic to propagate, got none")
			}
			if !isFermatPostConditionPanic(r) {
				t.Fatalf("expected post-condition sentinel, got %v", r)
			}
		}()
		_ = fermatPanicToError("len(z) > 2n+1", "SqrWithContext")
		t.Fatal("unreachable: helper must panic on sentinel")
	})
}

// TestMulRepanicsOnPostCondition forces a synthetic post-condition panic
// through Mul's deferred recover() and verifies that the panic propagates
// rather than being silently converted to an error. ADR-0002.
func TestMulRepanicsOnPostCondition(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected post-condition panic to propagate, got none")
		}
		if !isFermatPostConditionPanic(r) {
			t.Fatalf("expected post-condition sentinel, got %v", r)
		}
	}()

	// Replicate the deferred recover semantics of Mul without the FFT body.
	func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				if isFermatPostConditionPanic(r) {
					panic(r)
				}
				err = nil // would be wrapped as error in production
			}
		}()
		panic("fermat.Mul: unexpected carry after normalization")
	}()
}

// TestFermatRealPostConditionPanicIsClassified triggers a GENUINE
// post-condition panic through fermat.Mul instead of a synthetic string:
// denormalized operands with saturated words (a valid fermat keeps its top
// word in {0,1}) overflow the 2n+1-word product bound in the big.Int
// branch. Every other test in this file feeds the classifier strings taken
// from its own map, so rewording a panic() message in fermat.go without
// touching fermatPostConditionPanics kept them all green while demoting a
// real post-condition violation to an error (ADR-0002 violation). This test
// fails the moment the emitted message and the sentinel map drift apart,
// and pins that fermatPanicToError re-propagates the real panic.
func TestFermatRealPostConditionPanicIsClassified(t *testing.T) {
	t.Parallel()

	n := smallMulThreshold // big.Int branch, where the length bound is enforced
	x := make(fermat, n+1)
	for i := range x {
		x[i] = ^big.Word(0)
	}
	z := make(fermat, 2*n+2)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a post-condition panic from fermat.Mul on denormalized operands")
		}
		if !isFermatPostConditionPanic(r) {
			t.Fatalf("real post-condition panic %q is not recognized by the sentinel classifier; "+
				"the panic sites in fermat.go and fermatPostConditionPanics drifted apart", r)
		}
		// The ADR-0002 policy must re-propagate it, never convert to error.
		defer func() {
			if rr := recover(); rr == nil {
				t.Fatal("fermatPanicToError masked a real post-condition panic as an error")
			}
		}()
		_ = fermatPanicToError(r, "Mul")
	}()
	_ = z.Mul(x, x)
	t.Fatal("fermat.Mul returned normally on denormalized operands")
}
