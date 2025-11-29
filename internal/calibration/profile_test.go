package calibration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewProfile(t *testing.T) {
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

	expectedWordSize := 32 << (^uint(0) >> 63)
	if profile.WordSize != expectedWordSize {
		t.Errorf("WordSize = %d, want %d", profile.WordSize, expectedWordSize)
	}

	if profile.CalibratedAt.IsZero() {
		t.Error("CalibratedAt is zero")
	}
}

func TestProfileSaveLoad(t *testing.T) {
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
	loaded, err := LoadProfile(profilePath)
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

func TestProfileIsValid(t *testing.T) {
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

	// Nil profile
	var nilProfile *CalibrationProfile
	if nilProfile.IsValid() {
		t.Error("Expected nil profile to be invalid")
	}
}

func TestProfileIsStale(t *testing.T) {
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
	_, err := LoadProfile("/nonexistent/path/to/profile.json")
	if err == nil {
		t.Error("Expected error loading nonexistent profile")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
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

	_, err = LoadProfile(invalidPath)
	if err == nil {
		t.Error("Expected error loading invalid JSON")
	}
}

func TestLoadOrCreateProfile(t *testing.T) {
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

func TestProfileExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fibcalc_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, "profile.json")

	// Should not exist initially
	if ProfileExists(profilePath) {
		t.Error("Expected ProfileExists to return false for nonexistent file")
	}

	// Create the file
	profile := NewProfile()
	if err := profile.SaveProfile(profilePath); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	// Should exist now
	if !ProfileExists(profilePath) {
		t.Error("Expected ProfileExists to return true for existing file")
	}
}

func TestGetDefaultProfilePath(t *testing.T) {
	path := GetDefaultProfilePath()
	if path == "" {
		t.Error("GetDefaultProfilePath returned empty string")
	}

	// Should end with the default filename
	if filepath.Base(path) != DefaultProfileFileName {
		t.Errorf("Path %s doesn't end with %s", path, DefaultProfileFileName)
	}
}

