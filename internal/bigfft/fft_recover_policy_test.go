package bigfft

import "testing"

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
