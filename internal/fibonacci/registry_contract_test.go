package fibonacci

import (
	"strings"
	"testing"
)

// Contract tests for DefaultFactory.Register (audit API-02).
//
// Register rejects an empty name, a nil creator (which would defer the panic to
// whichever Get came first) and a duplicate name (which would silently swap out
// a built-in calculator). errcheck requires every call site to inspect the
// error; a Register that always returned nil would make that check pure
// ceremony.

func TestRegister_RejectsEmptyName(t *testing.T) {
	t.Parallel()
	f := NewDefaultFactory()

	err := f.Register("", func() CoreCalculator { return &FastDoublingCalculator{} })
	if err == nil {
		t.Fatal("Register accepted an empty name")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error does not name the cause: %v", err)
	}
}

func TestRegister_RejectsNilCreator(t *testing.T) {
	t.Parallel()
	f := NewDefaultFactory()

	err := f.Register("nilcreator", nil)
	if err == nil {
		t.Fatal("Register accepted a nil creator")
	}

	// The name must not have been recorded, or Get would panic later on a
	// creator that cannot be called.
	if _, getErr := f.Get("nilcreator"); getErr == nil {
		t.Error("a rejected registration is still resolvable through Get")
	}
}

func TestRegister_RejectsDuplicateAndKeepsTheOriginal(t *testing.T) {
	t.Parallel()
	f := NewDefaultFactory()

	original, err := f.Get("fast")
	if err != nil {
		t.Fatalf("built-in %q missing: %v", "fast", err)
	}

	err = f.Register("fast", func() CoreCalculator { return &MatrixExponentiationCalculator{} })
	if err == nil {
		t.Fatal("Register replaced a built-in calculator instead of refusing")
	}

	after, err := f.Get("fast")
	if err != nil {
		t.Fatalf("Get after a refused duplicate: %v", err)
	}
	if after != original {
		t.Error("a refused registration still disturbed the existing calculator")
	}
	if got := after.Name(); got != original.Name() {
		t.Errorf("calculator behind %q changed: %q -> %q", "fast", original.Name(), got)
	}
}

func TestRegister_AcceptsANewName(t *testing.T) {
	t.Parallel()
	f := NewDefaultFactory()

	if err := f.Register("extra", func() CoreCalculator { return &FastDoublingCalculator{} }); err != nil {
		t.Fatalf("Register rejected a valid new name: %v", err)
	}
	if _, err := f.Get("extra"); err != nil {
		t.Errorf("Get after a successful Register: %v", err)
	}

	names := f.List()
	if want := 4 + len(taggedRegistrations); len(names) != want {
		t.Errorf("List = %v, want the built-ins (%d under this build) plus %q", names, want-1, "extra")
	}
	// List documents a sorted result; "extra" sorts between "fft" and "fast"?
	// Assert the property, not a hand-computed order.
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("List is not sorted: %v", names)
			break
		}
	}
}

// The three built-ins must all be present and distinct; NewDefaultFactory now
// panics rather than registering a subset in silence. A -tags gmp build adds
// "gmp" through taggedRegistrations (EVAL-23), and the expectation follows.
func TestNewDefaultFactory_RegistersEveryBuiltin(t *testing.T) {
	t.Parallel()
	f := NewDefaultFactory()

	want := []string{"fast", "fft", "matrix"}
	if len(taggedRegistrations) > 0 {
		want = []string{"fast", "fft", "gmp", "matrix"}
	}
	got := f.List()
	if len(got) != len(want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("List = %v, want %v (sorted)", got, want)
		}
	}

	all := f.GetAll()
	if len(all) != len(want) {
		t.Errorf("GetAll returned %d calculators, want %d", len(all), len(want))
	}
	for _, name := range want {
		if all[name] == nil {
			t.Errorf("GetAll is missing %q", name)
		}
	}
}

// TestFactory.List claims to be interchangeable with DefaultFactory.List, which
// documents a sorted result. It returned map order until audit API-01.
func TestTestFactory_ListIsSorted(t *testing.T) {
	t.Parallel()
	f := NewTestFactory(map[string]Calculator{
		"zulu":  &MockCalculator{},
		"alpha": &MockCalculator{},
		"mike":  &MockCalculator{},
	})

	got := f.List()
	want := []string{"alpha", "mike", "zulu"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("List = %v, want %v", got, want)
		}
	}
}
