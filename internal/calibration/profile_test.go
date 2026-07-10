// Package calibration_test exercises internal/calibration's exported API
// only. loadProfile (unexported) is reached transitively through
// LoadOrCreateProfile in every case below: for a profile written by this
// same process, IsValid() is always true (NewProfile stamps the current
// hardware fields), so LoadOrCreateProfile's (profile, ok) contract
// exercises the exact same read-and-parse code path with equal assertion
// strength as calling loadProfile directly.
package calibration_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agbruneau/FibGo/internal/calibration"
)

func TestNewProfile(t *testing.T) {
	t.Parallel()
	profile := calibration.NewProfile()

	if profile == nil {
		t.Fatal("NewProfile returned nil")
	}
	if profile.NumCPU != runtime.NumCPU() {
		t.Errorf("NumCPU = %d, want %d", profile.NumCPU, runtime.NumCPU())
	}
	if profile.GOARCH != runtime.GOARCH {
		t.Errorf("GOARCH = %s, want %s", profile.GOARCH, runtime.GOARCH)
	}
	if profile.GOOS != runtime.GOOS {
		t.Errorf("GOOS = %s, want %s", profile.GOOS, runtime.GOOS)
	}
	if profile.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %s, want %s", profile.GoVersion, runtime.Version())
	}
	if profile.ProfileVersion != calibration.CurrentProfileVersion {
		t.Errorf("ProfileVersion = %d, want %d", profile.ProfileVersion, calibration.CurrentProfileVersion)
	}
	if profile.CPUHeuristicKey == "" {
		t.Error("CPUHeuristicKey should be non-empty")
	}

	expectedWordSize := 32 << (^uint(0) >> 63)
	if profile.WordSize != expectedWordSize {
		t.Errorf("WordSize = %d, want %d", profile.WordSize, expectedWordSize)
	}
	if profile.CalibratedAt.IsZero() {
		t.Error("CalibratedAt is zero")
	}
}

func TestProfileSaveLoad(t *testing.T) {
	t.Parallel()
	profilePath := filepath.Join(t.TempDir(), "test_profile.json")

	original := calibration.NewProfile()
	original.OptimalParallelThreshold = 4096
	original.OptimalFFTThreshold = 1000000
	original.OptimalStrassenThreshold = 256
	original.CalibrationN = 10000000
	original.CalibrationTime = "1m30s"

	if err := original.SaveProfile(profilePath); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Fatal("Profile file was not created")
	}

	loaded, ok := calibration.LoadOrCreateProfile(profilePath)
	if !ok {
		t.Fatal("LoadOrCreateProfile should report the just-saved profile as loaded")
	}
	if loaded.OptimalParallelThreshold != original.OptimalParallelThreshold {
		t.Errorf("OptimalParallelThreshold = %d, want %d",
			loaded.OptimalParallelThreshold, original.OptimalParallelThreshold)
	}
	if loaded.OptimalFFTThreshold != original.OptimalFFTThreshold {
		t.Errorf("OptimalFFTThreshold = %d, want %d",
			loaded.OptimalFFTThreshold, original.OptimalFFTThreshold)
	}
	if loaded.OptimalStrassenThreshold != original.OptimalStrassenThreshold {
		t.Errorf("OptimalStrassenThreshold = %d, want %d",
			loaded.OptimalStrassenThreshold, original.OptimalStrassenThreshold)
	}
	if loaded.NumCPU != original.NumCPU {
		t.Errorf("NumCPU = %d, want %d", loaded.NumCPU, original.NumCPU)
	}
}

// TestSaveProfile_AtomicNoPartialFile verifies that SaveProfile writes the
// profile atomically: no temporary "*.tmp*" file is left behind in the
// target directory, the final file is always a complete, reloadable JSON
// document (never a truncated prefix), and overwriting an existing valid
// profile keeps a complete document at every observable point.
func TestSaveProfile_AtomicNoPartialFile(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profile.json")

	first := calibration.NewProfile()
	first.OptimalParallelThreshold = 4096
	first.OptimalFFTThreshold = 1000000
	first.OptimalStrassenThreshold = 256
	first.CalibrationN = 10000000
	first.CalibrationTime = "1m30s"
	if err := first.SaveProfile(profilePath); err != nil {
		t.Fatalf("SaveProfile (first) failed: %v", err)
	}
	assertNoTempResidue(t, tmpDir)
	assertReloadableMatches(t, profilePath, first)

	// Overwrite with a different profile: the final file must again be a
	// complete document (atomic rename, not an in-place partial truncation).
	second := calibration.NewProfile()
	second.OptimalParallelThreshold = 65536
	second.OptimalFFTThreshold = 2000000
	second.OptimalStrassenThreshold = 512
	second.CalibrationN = 99999999
	second.CalibrationTime = "5m00s"
	if err := second.SaveProfile(profilePath); err != nil {
		t.Fatalf("SaveProfile (overwrite) failed: %v", err)
	}
	assertNoTempResidue(t, tmpDir)
	assertReloadableMatches(t, profilePath, second)
}

// assertNoTempResidue fails if any "*.tmp*" file remains in dir.
func assertNoTempResidue(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("Temporary file residue found in %s: %s", dir, e.Name())
		}
	}
}

// assertReloadableMatches reloads the profile at path via LoadOrCreateProfile
// and verifies the persisted fields match want exactly (i.e. the file is a
// complete, non-truncated JSON document).
func assertReloadableMatches(t *testing.T, path string, want *calibration.CalibrationProfile) {
	t.Helper()
	got, ok := calibration.LoadOrCreateProfile(path)
	if !ok {
		t.Fatalf("LoadOrCreateProfile(%s) reported not-loaded (file truncated, corrupt, or hardware-mismatched?)", path)
	}
	if got.OptimalParallelThreshold != want.OptimalParallelThreshold {
		t.Errorf("OptimalParallelThreshold = %d, want %d",
			got.OptimalParallelThreshold, want.OptimalParallelThreshold)
	}
	if got.OptimalFFTThreshold != want.OptimalFFTThreshold {
		t.Errorf("OptimalFFTThreshold = %d, want %d",
			got.OptimalFFTThreshold, want.OptimalFFTThreshold)
	}
	if got.OptimalStrassenThreshold != want.OptimalStrassenThreshold {
		t.Errorf("OptimalStrassenThreshold = %d, want %d",
			got.OptimalStrassenThreshold, want.OptimalStrassenThreshold)
	}
	if got.CalibrationN != want.CalibrationN {
		t.Errorf("CalibrationN = %d, want %d", got.CalibrationN, want.CalibrationN)
	}
	if got.CalibrationTime != want.CalibrationTime {
		t.Errorf("CalibrationTime = %q, want %q", got.CalibrationTime, want.CalibrationTime)
	}
	if got.ProfileVersion != want.ProfileVersion {
		t.Errorf("ProfileVersion = %d, want %d", got.ProfileVersion, want.ProfileVersion)
	}
}

// TestSaveProfile_NeverObservablyTruncated asserts the core A-11 invariant:
// while SaveProfile repeatedly rewrites an already-valid profile, a
// concurrent reader of the SAME path must never observe the file content
// as a syntactically truncated/partial JSON document -- it must always
// parse either the complete old document or the complete new one.
//
// With a non-atomic implementation (os.WriteFile truncates the target to 0
// bytes before refilling it) the reader observes an empty/partial file and
// json.Unmarshal fails with "unexpected end of JSON input". With the
// atomic implementation (sibling temp file + os.Rename) the swap is
// all-or-nothing.
//
// Portability note: on Windows os.Rename over a target that another handle
// has open can transiently fail with a sharing/access error on EITHER
// side. That is NOT a content-truncation symptom (the bytes on disk are
// always a complete document). Both sides treat such transient OS access
// errors as retryable: the reader retries the read, and the writer retries
// the same SaveProfile instead of failing the test. Only a parse failure
// on successfully-read bytes counts as the A-11 defect.
func TestSaveProfile_NeverObservablyTruncated(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "profile.json")

	seed := calibration.NewProfile()
	seed.OptimalParallelThreshold = 4096
	if err := seed.SaveProfile(profilePath); err != nil {
		t.Fatalf("seed SaveProfile failed: %v", err)
	}

	const rewrites = 80
	done := make(chan struct{})
	var truncErr error
	var errOnce sync.Once

	var rwg sync.WaitGroup
	rwg.Add(1)
	go func() {
		defer rwg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			data, rerr := os.ReadFile(profilePath)
			if rerr != nil {
				// Transient OS access/sharing error (Windows rename window) or
				// the brief moment the path is being replaced: not a content
				// truncation. Retry.
				continue
			}
			if !json.Valid(data) {
				// Bytes were read successfully but are an incomplete JSON
				// document: this is exactly the A-11 corruption.
				errOnce.Do(func() {
					truncErr = errors.New("reader observed syntactically truncated JSON: " + string(data))
				})
				return
			}
		}
	}()

	for i := 0; i < rewrites; i++ {
		p := calibration.NewProfile()
		p.OptimalParallelThreshold = 1000 + i
		writeErr := p.SaveProfile(profilePath)
		if writeErr == nil {
			continue
		}
		// Symmetric to the reader: on Windows the atomic rename can
		// transiently fail with a sharing/access error while the
		// concurrent reader holds the target open. That is not a
		// content-truncation symptom, so retry the same rewrite instead of
		// failing.
		if runtime.GOOS == "windows" && errors.Is(writeErr, fs.ErrPermission) {
			i--
			continue
		}
		close(done)
		rwg.Wait()
		t.Fatalf("rewrite SaveProfile(%d) failed: %v", i, writeErr)
	}
	close(done)
	rwg.Wait()

	if truncErr != nil {
		t.Fatalf("non-atomic write detected: %v", truncErr)
	}
	assertNoTempResidue(t, tmpDir)
	if _, ok := calibration.LoadOrCreateProfile(profilePath); !ok {
		t.Fatal("final LoadOrCreateProfile reported not-loaded (corrupt file?)")
	}
}

func TestProfileIsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*calibration.CalibrationProfile)
		want   bool
	}{
		{"valid for current hardware", func(*calibration.CalibrationProfile) {}, true},
		{"wrong CPU count", func(p *calibration.CalibrationProfile) { p.NumCPU = 999 }, false},
		{"wrong architecture", func(p *calibration.CalibrationProfile) { p.GOARCH = "invalid_arch" }, false},
		{"wrong word size", func(p *calibration.CalibrationProfile) { p.WordSize = 16 }, false},
		{"wrong version", func(p *calibration.CalibrationProfile) { p.ProfileVersion = 999 }, false},
		{"wrong CPU heuristic key", func(p *calibration.CalibrationProfile) { p.CPUHeuristicKey = "mismatched-key" }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := calibration.NewProfile()
			tc.mutate(p)
			if got := p.IsValid(); got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}

	var nilProfile *calibration.CalibrationProfile
	if nilProfile.IsValid() {
		t.Error("Expected nil profile to be invalid")
	}
}

func TestProfileIsStale(t *testing.T) {
	t.Parallel()
	profile := calibration.NewProfile()

	if profile.IsStale(time.Hour) {
		t.Error("Expected fresh profile to not be stale")
	}

	profile.CalibratedAt = time.Now().Add(-2 * time.Hour)
	if !profile.IsStale(time.Hour) {
		t.Error("Expected old profile to be stale")
	}

	var nilProfile *calibration.CalibrationProfile
	if !nilProfile.IsStale(time.Hour) {
		t.Error("Expected nil profile to be stale")
	}
}

func TestProfileString(t *testing.T) {
	t.Parallel()
	profile := calibration.NewProfile()
	profile.OptimalParallelThreshold = 100
	profile.OptimalFFTThreshold = 200
	profile.OptimalStrassenThreshold = 300
	profile.CalibrationN = 1000
	profile.CalibrationTime = "1s"

	str := profile.String()
	if len(str) < 50 {
		t.Errorf("String() seems too short: %s", str)
	}
	for _, want := range []string{"Parallel: 100 bits", "FFT: 200 bits", "Strassen: 300 bits", "CalibrationProfile{"} {
		if !strings.Contains(str, want) {
			t.Errorf("String() missing %q, got: %s", want, str)
		}
	}
}

func TestProfile_SaveProfile_Error(t *testing.T) {
	t.Parallel()
	p := calibration.NewProfile()
	// Try to save to a directory that doesn't exist/invalid path.
	if err := p.SaveProfile("\x00invalid/path/profile.json"); err == nil {
		t.Error("Expected error saving to invalid path")
	}
}

func TestLoadNonExistentProfile(t *testing.T) {
	t.Parallel()
	_, ok := calibration.LoadOrCreateProfile("/nonexistent/path/to/profile.json")
	if ok {
		t.Error("Expected LoadOrCreateProfile to report not-loaded for a nonexistent profile")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	t.Parallel()
	invalidPath := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, ok := calibration.LoadOrCreateProfile(invalidPath)
	if ok {
		t.Error("Expected LoadOrCreateProfile to report not-loaded for invalid JSON")
	}
}

func TestLoadOrCreateProfile(t *testing.T) {
	t.Parallel()
	profilePath := filepath.Join(t.TempDir(), "profile.json")

	// First call should create a new profile.
	profile, loaded := calibration.LoadOrCreateProfile(profilePath)
	if loaded {
		t.Error("Expected loaded to be false for nonexistent file")
	}
	if profile == nil {
		t.Fatal("Expected profile to not be nil")
	}

	profile.OptimalParallelThreshold = 8192
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Second call should load the existing profile.
	profile2, loaded2 := calibration.LoadOrCreateProfile(profilePath)
	if !loaded2 {
		t.Error("Expected loaded to be true for existing file")
	}
	if profile2.OptimalParallelThreshold != 8192 {
		t.Errorf("Loaded profile has wrong threshold: %d", profile2.OptimalParallelThreshold)
	}
}

func TestGetDefaultProfilePath(t *testing.T) {
	t.Parallel()
	path := calibration.GetDefaultProfilePath()
	if path == "" {
		t.Error("GetDefaultProfilePath returned empty string")
	}
	if filepath.Base(path) != calibration.DefaultProfileFileName {
		t.Errorf("Path %s doesn't end with %s", path, calibration.DefaultProfileFileName)
	}
}
