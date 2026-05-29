package bigfft

import (
	"strings"
	"testing"
)

// TestFermatPanicSites documents and verifies every pre-condition panic
// emitted by fermat.go. Each case constructs an operand-size mismatch and
// asserts that the documented panic message fires. Audit-PRD E10-R1.
//
// Post-condition panics ("len(z) > 2n+1", "unexpected carry after
// normalization") are exercised by TestFermatPostConditionPanicClassifier
// in fft_recover_policy_test.go; this file focuses on the *caller-fault*
// pre-conditions that should remain panics (they signal a bug above the
// fermat layer).
func TestFermatPanicSites(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		fn   func()
		want string
	}{
		{
			name: "Shift: len(z) != len(x)",
			fn: func() {
				z := make(fermat, 5)
				x := make(fermat, 7)
				z.Shift(x, 1)
			},
			want: "len(z) != len(x) in Shift",
		},
		{
			name: "Add: len(z) != len(x)",
			fn: func() {
				z := make(fermat, 5)
				x := make(fermat, 7)
				y := make(fermat, 7)
				z.Add(x, y)
			},
			want: "Add: len(z) != len(x)",
		},
		{
			name: "Sub: len(z) != len(x)",
			fn: func() {
				z := make(fermat, 5)
				x := make(fermat, 7)
				y := make(fermat, 7)
				z.Sub(x, y)
			},
			want: "fermat.Sub: len(z) != len(x)",
		},
		{
			name: "Mul: len(x) != len(y)",
			fn: func() {
				z := make(fermat, 7)
				x := make(fermat, 5)
				y := make(fermat, 7)
				_ = z.Mul(x, y)
			},
			want: "Mul: len(x) != len(y)",
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected panic %q, got none", tc.want)
				}
				msg, ok := r.(string)
				if !ok {
					t.Fatalf("expected string panic %q, got %T %v", tc.want, r, r)
				}
				if !strings.Contains(msg, tc.want) {
					t.Errorf("panic message %q does not contain %q", msg, tc.want)
				}
			}()
			tc.fn()
		})
	}
}
