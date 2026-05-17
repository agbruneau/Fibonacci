package bigfft

import (
	"strings"
	"testing"
)

// TestNTransformPreconditionReturnsError is the [A-07] regression test.
//
// (*Poly).NTransform is a public API that already returns (PolValues,
// error), yet it panicked on the precondition len(p.A) >= 1<<k OUTSIDE
// any recover() — so a caller passing a too-large coefficient slice got
// an uncatchable crash instead of the error the signature promises.
//
// The fix routes the precondition to a returned error. The panic must be
// gone and a descriptive error returned with a zero-value PolValues.
func TestNTransformPreconditionReturnsError(t *testing.T) {
	t.Parallel()

	// k = 2 => 1<<k == 4. Provide 5 coefficients so len(p.A) >= 1<<k.
	p := &Poly{
		K: 2,
		M: 1,
		A: []nat{{1}, {2}, {3}, {4}, {5}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("[A-07] NTransform panicked instead of returning an "+
				"error on precondition violation: %v", r)
		}
	}()

	pv, err := p.NTransform(8)
	if err == nil {
		t.Fatal("[A-07] expected an error when len(p.A) >= 1<<k, got nil")
	}
	if !strings.Contains(err.Error(), "NTransform") {
		t.Fatalf("[A-07] error should identify NTransform: %v", err)
	}
	if pv.Values != nil || pv.K != 0 || pv.N != 0 {
		t.Fatalf("[A-07] expected zero-value PolValues on error, got %+v", pv)
	}

	// Sanity: a well-formed input (len(p.A) < 1<<k) must still succeed.
	good := &Poly{K: 2, M: 1, A: []nat{{1}, {2}}}
	if _, err := good.NTransform(8); err != nil {
		t.Fatalf("[A-07] well-formed NTransform unexpectedly failed: %v", err)
	}
}
