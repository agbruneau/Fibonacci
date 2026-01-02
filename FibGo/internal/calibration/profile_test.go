package calibration

import (
	"os"
	"path/filepath"
	"runtime"
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
	_, err := LoadProfile("/nonexistent/path/to/profile.json")
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

	_, err = LoadProfile(invalidPath)
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

func TestProfileExists(t *testing.T) {
	t.Parallel()
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

func TestProfileRanges(t *testing.T) {
	t.Parallel()
	profile := NewProfile()
	profile.OptimalFFTThreshold = 1000
	profile.OptimalParallelThreshold = 1000
	profile.OptimalStrassenThreshold = 1000
	profile.InitializeDefaultRanges()

	if len(profile.ThresholdsByRange) == 0 {
		t.Error("InitializeDefaultRanges should add ranges")
	}

	// Test GetThresholdsForN
	// With default ranges, it should return defaults (which we just set)
	fft, par, strassen := profile.GetThresholdsForN(50000)
	if fft != 1000 || par != 1000 || strassen != 1000 {
		t.Errorf("GetThresholdsForN = %d, %d, %d; want 1000, 1000, 1000", fft, par, strassen)
	}

	// Add a specific range
	r := RangeThresholds{
		MinN:              100000,
		MaxN:              200000,
		FFTThreshold:      123,
		ParallelThreshold: 456,
		StrassenThreshold: 789,
		ConfidenceScore:   1.0,
		MeasurementCount:  10,
	}
	profile.AddRangeThresholds(r)

	// Test GetThresholdsForN for the new range
	fft, par, strassen = profile.GetThresholdsForN(150000)
	if fft != 123 || par != 456 || strassen != 789 {
		t.Errorf("GetThresholdsForN = %d, %d, %d; want 123, 456, 789", fft, par, strassen)
	}
}

func TestAddRangeThresholds(t *testing.T) {
	t.Parallel()
	profile := NewProfile()

	r1 := RangeThresholds{
		MinN:              100,
		MaxN:              200,
		FFTThreshold:      1000,
		ParallelThreshold: 1000,
		ConfidenceScore:   0.5,
		MeasurementCount:  1,
	}
	profile.AddRangeThresholds(r1)

	// Add same range with different values to test merging
	r2 := RangeThresholds{
		MinN:              100,
		MaxN:              200,
		FFTThreshold:      2000,
		ParallelThreshold: 2000,
		ConfidenceScore:   0.5,
		MeasurementCount:  1,
	}
	profile.AddRangeThresholds(r2)

	fft, par, _ := profile.GetThresholdsForN(150)
	// Weighted average: (1000*1 + 2000*1) / 2 = 1500
	if fft != 1500 || par != 1500 {
		t.Errorf("Merged thresholds = %d, %d; want 1500, 1500", fft, par)
	}
}
