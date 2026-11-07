package testutil

import (
	"os"
	"testing"
)

// Unsetenv removes key for the duration of t and restores the previous state
// when t finishes.
//
// It exists because the standard library has t.Setenv but no t.Unsetenv, and
// for a variable read through os.LookupEnv — NO_COLOR is the case here, where
// presence alone disables color — "absent" and "set to the empty string" are
// different states. t.Setenv(key, "") is therefore not a substitute.
//
// The t.Setenv call is what registers the restore (it records whether the
// variable existed and puts it back on cleanup); the Unsetenv that follows
// produces the state under test. Like t.Setenv, this cannot be used from a
// parallel test — which is the point: a test that mutates the process
// environment must not run alongside one that reads it (audit TST-02).
func Unsetenv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
}
