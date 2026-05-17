package calibration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewProfile(t *testing.T) {
	t.Parallel()
	profile := NewProfile()

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

	if profile.ProfileVersion != CurrentProfileVersion {
		t.Errorf("ProfileVersion = %d, want %d", profile.ProfileVersion, CurrentProfileVersion)
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
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "fibcalc_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, "test_profile.json")

	// Create and save a profile
	original := NewProfile()
	original.OptimalParallelThreshold = 4096
	original.OptimalFFTThreshold = 1000000
	original.OptimalStrassenThreshold = 256
	original.CalibrationN = 10000000
	original.CalibrationTime = "1m30s"

	if err := original.SaveProfile(profilePath); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Fatal("Profile file was not created")
	}

	// Load the profile
	loaded, err := loadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	// Verify loaded values
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
// profile atomically: no temporary "*.tmp*" file is left behind in the target
// directory, the final file is always a complete, reloadable JSON document
// (never a truncated prefix), and overwriting an existing valid profile keeps
// a complete document at every observable point.
func TestSaveProfile_AtomicNoPartialFile(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "fibcalc_atomic_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, "profile.json")

	// First write.
	first := NewProfile()
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
	second := NewProfile()
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

// assertReloadableMatches reloads the profile at path through the existing
// loader and verifies the persisted fields match want exactly (i.e. the file
// is a complete, non-truncated JSON document).
func assertReloadableMatches(t *testing.T, path string, want *CalibrationProfile) {
	t.Helper()
	got, err := loadProfile(path)
	if err != nil {
		t.Fatalf("loadProfile(%s) failed (file truncated or corrupt?): %v", path, err)
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
// while SaveProfile repeatedly rewrites an already-valid profile, a concurrent
// reader of the SAME path must never observe the file content as a
// syntactically truncated/partial JSON document — it must always parse either
// the complete old document or the complete new one.
//
// With the non-atomic implementation (os.WriteFile truncates the target to 0
// bytes before refilling it) the reader observes an empty/partial file and
// json.Unmarshal fails with "unexpected end of JSON input". With the atomic
// implementation (sibling temp file + os.Rename) the swap is all-or-nothing.
//
// Portability note: on Windows os.Rename over a target that another handle has
// open can transiently fail with a sharing/access error on EITHER side. That
// is NOT a content-truncation symptom (the bytes on disk are always a complete
// document); the reader treats such transient OS access errors as retryable.
// Only a parse failure on successfully-read bytes counts as the A-11 defect.
func TestSaveProfile_NeverObservablyTruncated(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "fibcalc_atomic_obs_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, "profile.json")

	seed := NewProfile()
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
		p := NewProfile()
		p.OptimalParallelThreshold = 1000 + i
		if err := p.SaveProfile(profilePath); err != nil {
			close(done)
			rwg.Wait()
			t.Fatalf("rewrite SaveProfile(%d) failed: %v", i, err)
		}
	}
	close(done)
	rwg.Wait()

	if truncErr != nil {
		t.Fatalf("non-atomic write detected: %v", truncErr)
	}

	assertNoTempResidue(t, tmpDir)
	if _, err := loadProfile(profilePath); err != nil {
		t.Fatalf("final loadProfile failed (corrupt file?): %v", err)
	}
}

func TestProfileIsValid(t *testing.T) {
	t.Parallel()
	// Valid profile for current hardware
	valid := NewProfile()
	if !valid.IsValid() {
		t.Error("Expected newly created profile to be valid")
	}

	// Invalid: wrong CPU count
	wrongCPU := NewProfile()
	wrongCPU.NumCPU = 999
	if wrongCPU.IsValid() {
		t.Error("Expected profile with wrong CPU count to be invalid")
	}

	// Invalid: wrong architecture
	wrongArch := NewProfile()
	wrongArch.GOARCH = "invalid_arch"
	if wrongArch.IsValid() {
		t.Error("Expected profile with wrong GOARCH to be invalid")
	}

	// Invalid: wrong word size
	wrongWordSize := NewProfile()
	wrongWordSize.WordSize = 16
	if wrongWordSize.IsValid() {
		t.Error("Expected profile with wrong word size to be invalid")
	}

	// Invalid: wrong version
	wrongVersion := NewProfile()
	wrongVersion.ProfileVersion = 999
	if wrongVersion.IsValid() {
		t.Error("Expected profile with wrong version to be invalid")
	}

	// Invalid: wrong CPU heuristic key (SIMD / arch class)
	wrongHeur := NewProfile()
	wrongHeur.CPUHeuristicKey = "mismatched-key"
	if wrongHeur.IsValid() {
		t.Error("Expected profile with wrong CPUHeuristicKey to be invalid")
	}

	// Nil profile
	var nilProfile *CalibrationProfile
	if nilProfile.IsValid() {
		t.Error("Expected nil profile to be invalid")
	}
}

func TestProfileIsStale(t *testing.T) {
	t.Parallel()
	profile := NewProfile()

	// Fresh profile should not be stale
	if profile.IsStale(time.Hour) {
		t.Error("Expected fresh profile to not be stale")
	}

	// Old profile should be stale
	profile.CalibratedAt = time.Now().Add(-2 * time.Hour)
	if !profile.IsStale(time.Hour) {
		t.Error("Expected old profile to be stale")
	}

	// Nil profile should be stale
	var nilProfile *CalibrationProfile
	if !nilProfile.IsStale(time.Hour) {
		t.Error("Expected nil profile to be stale")
	}
}

func TestProfileString(t *testing.T) {
	t.Parallel()
	profile := NewProfile()
	profile.OptimalParallelThreshold = 4096
	profile.OptimalFFTThreshold = 1000000
	profile.OptimalStrassenThreshold = 256

	str := profile.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check it contains key information
	if len(str) < 50 {
		t.Errorf("String() seems too short: %s", str)
	}
}

func TestLoadNonExistentProfile(t *testing.T) {
	t.Parallel()
	_, err := loadProfile("/nonexistent/path/to/profile.json")
	if err == nil {
		t.Error("Expected error loading nonexistent profile")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "fibcalc_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file with invalid JSON
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	_, err = loadProfile(invalidPath)
	if err == nil {
		t.Error("Expected error loading invalid JSON")
	}
}

func TestLoadOrCreateProfile(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "fibcalc_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, "profile.json")

	// First call should create new profile
	profile, loaded := LoadOrCreateProfile(profilePath)
	if loaded {
		t.Error("Expected loaded to be false for nonexistent file")
	}
	if profile == nil {
		t.Fatal("Expected profile to not be nil")
	}

	// Save the profile
	profile.OptimalParallelThreshold = 8192
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Second call should load existing profile
	profile2, loaded2 := LoadOrCreateProfile(profilePath)
	if !loaded2 {
		t.Error("Expected loaded to be true for existing file")
	}
	if profile2.OptimalParallelThreshold != 8192 {
		t.Errorf("Loaded profile has wrong threshold: %d", profile2.OptimalParallelThreshold)
	}
}

func TestGetDefaultProfilePath(t *testing.T) {
	t.Parallel()
	path := GetDefaultProfilePath()
	if path == "" {
		t.Error("GetDefaultProfilePath returned empty string")
	}

	// Should end with the default filename
	if filepath.Base(path) != DefaultProfileFileName {
		t.Errorf("Path %s doesn't end with %s", path, DefaultProfileFileName)
	}
}
