// Package calibration provides performance calibration for the Fibonacci calculator.
// This file implements calibration profile persistence.
package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// CalibrationProfile stores the results of a calibration run.
// It captures both the optimal thresholds and the hardware context
// to allow validation of cached results.
type CalibrationProfile struct {
	// Hardware identification
	CPUModel  string `json:"cpu_model"`
	NumCPU    int    `json:"num_cpu"`
	GOARCH    string `json:"goarch"`
	GOOS      string `json:"goos"`
	GoVersion string `json:"go_version"`
	WordSize  int    `json:"word_size"` // 32 or 64

	// Calibrated thresholds (default/fallback values)
	OptimalParallelThreshold int `json:"optimal_parallel_threshold"`
	OptimalFFTThreshold      int `json:"optimal_fft_threshold"`
	OptimalStrassenThreshold int `json:"optimal_strassen_threshold"`

	// Thresholds by N range for more accurate calibration
	ThresholdsByRange []RangeThresholds `json:"thresholds_by_range,omitempty"`

	// Calibration metadata
	CalibratedAt    time.Time `json:"calibrated_at"`
	CalibrationN    uint64    `json:"calibration_n"`
	CalibrationTime string    `json:"calibration_time"`

	// Version for forward compatibility
	ProfileVersion int `json:"profile_version"`
}

// RangeThresholds stores optimal thresholds for a specific range of N values.
// This allows for more accurate threshold selection based on the problem size.
type RangeThresholds struct {
	// MinN is the minimum N value (inclusive) for this range
	MinN uint64 `json:"min_n"`
	// MaxN is the maximum N value (inclusive) for this range
	MaxN uint64 `json:"max_n"`
	// FFTThreshold is the optimal FFT threshold for this range
	FFTThreshold int `json:"fft_threshold"`
	// ParallelThreshold is the optimal parallel threshold for this range
	ParallelThreshold int `json:"parallel_threshold"`
	// StrassenThreshold is the optimal Strassen threshold for this range
	StrassenThreshold int `json:"strassen_threshold,omitempty"`
	// ConfidenceScore indicates the reliability of these thresholds (0-1)
	ConfidenceScore float64 `json:"confidence_score"`
	// MeasurementCount is the number of measurements used to derive these thresholds
	MeasurementCount int `json:"measurement_count"`
}

const (
	// CurrentProfileVersion is the current version of the profile format.
	// Increment this when making breaking changes to the profile structure.
	CurrentProfileVersion = 2

	// DefaultProfileFileName is the default name for the calibration profile file.
	DefaultProfileFileName = ".fibcalc_calibration.json"
)

// Predefined N ranges for calibration
var DefaultNRanges = []struct {
	MinN, MaxN uint64
	Label      string
}{
	{0, 100000, "small"},            // N < 100K
	{100000, 1000000, "medium"},     // 100K <= N < 1M
	{1000000, 10000000, "large"},    // 1M <= N < 10M
	{10000000, 100000000, "xlarge"}, // 10M <= N < 100M
	{100000000, ^uint64(0), "huge"}, // N >= 100M
}

// GetDefaultProfilePath returns the default path for the calibration profile.
// It uses the user's home directory if available, otherwise the current directory.
func GetDefaultProfilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultProfileFileName
	}
	return filepath.Join(home, DefaultProfileFileName)
}

// NewProfile creates a new CalibrationProfile with current hardware info.
func NewProfile() *CalibrationProfile {
	return &CalibrationProfile{
		CPUModel:       getCPUModel(),
		NumCPU:         runtime.NumCPU(),
		GOARCH:         runtime.GOARCH,
		GOOS:           runtime.GOOS,
		GoVersion:      runtime.Version(),
		WordSize:       32 << (^uint(0) >> 63), // 32 or 64
		CalibratedAt:   time.Now(),
		ProfileVersion: CurrentProfileVersion,
	}
}

// getCPUModel attempts to get a CPU model identifier.
// This is platform-specific and may return a generic value.
func getCPUModel() string {
	// On most systems, we can use GOARCH + NumCPU as a reasonable identifier
	// A more sophisticated implementation could read /proc/cpuinfo on Linux
	// or use syscalls on other platforms
	return fmt.Sprintf("%s-%d-cores", runtime.GOARCH, runtime.NumCPU())
}

// LoadProfile loads a calibration profile from the specified path.
// Returns nil and an error if the file doesn't exist or can't be parsed.
func LoadProfile(path string) (*CalibrationProfile, error) {
	if path == "" {
		path = GetDefaultProfilePath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	var profile CalibrationProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

// SaveProfile saves the calibration profile to the specified path.
// If path is empty, uses the default profile path.
func (p *CalibrationProfile) SaveProfile(path string) error {
	if path == "" {
		path = GetDefaultProfilePath()
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// IsValid checks if the profile is valid for the current hardware.
// A profile is considered valid if:
// - The profile version matches
// - The number of CPUs matches
// - The architecture matches
// - The word size matches
func (p *CalibrationProfile) IsValid() bool {
	if p == nil {
		return false
	}

	// Check version compatibility
	if p.ProfileVersion != CurrentProfileVersion {
		return false
	}

	// Check hardware compatibility
	if p.NumCPU != runtime.NumCPU() {
		return false
	}

	if p.GOARCH != runtime.GOARCH {
		return false
	}

	wordSize := 32 << (^uint(0) >> 63)
	if p.WordSize != wordSize {
		return false
	}

	return true
}

// IsStale checks if the profile is older than the given duration.
// This can be used to trigger re-calibration after a certain period.
func (p *CalibrationProfile) IsStale(maxAge time.Duration) bool {
	if p == nil {
		return true
	}
	return time.Since(p.CalibratedAt) > maxAge
}

// String returns a human-readable summary of the profile.
func (p *CalibrationProfile) String() string {
	if p == nil {
		return "<nil profile>"
	}

	rangeInfo := ""
	if len(p.ThresholdsByRange) > 0 {
		rangeInfo = fmt.Sprintf(", Ranges: %d", len(p.ThresholdsByRange))
	}

	return fmt.Sprintf(
		"CalibrationProfile{CPU: %s, Parallel: %d bits, FFT: %d bits, Strassen: %d bits%s, Calibrated: %s}",
		p.CPUModel,
		p.OptimalParallelThreshold,
		p.OptimalFFTThreshold,
		p.OptimalStrassenThreshold,
		rangeInfo,
		p.CalibratedAt.Format(time.RFC3339),
	)
}

// GetThresholdsForN returns the optimal thresholds for a given N value.
// If a matching range is found with sufficient confidence, those thresholds are returned.
// Otherwise, the default thresholds are returned.
func (p *CalibrationProfile) GetThresholdsForN(n uint64) (fft, parallel, strassen int) {
	if p == nil {
		return 0, 0, 0
	}

	// Search for a matching range with good confidence
	for _, r := range p.ThresholdsByRange {
		if n >= r.MinN && n <= r.MaxN && r.ConfidenceScore >= 0.5 {
			fft = r.FFTThreshold
			parallel = r.ParallelThreshold
			strassen = r.StrassenThreshold
			if strassen == 0 {
				strassen = p.OptimalStrassenThreshold
			}
			return fft, parallel, strassen
		}
	}

	// Fall back to default thresholds
	return p.OptimalFFTThreshold, p.OptimalParallelThreshold, p.OptimalStrassenThreshold
}

// AddRangeThresholds adds or updates thresholds for a specific N range.
// If a range with the same bounds exists, it is updated with the new values
// using a weighted average based on measurement counts.
func (p *CalibrationProfile) AddRangeThresholds(r RangeThresholds) {
	// Look for existing range with same bounds
	for i, existing := range p.ThresholdsByRange {
		if existing.MinN == r.MinN && existing.MaxN == r.MaxN {
			// Update existing range with weighted average
			totalCount := existing.MeasurementCount + r.MeasurementCount
			if totalCount > 0 {
				existingWeight := float64(existing.MeasurementCount) / float64(totalCount)
				newWeight := float64(r.MeasurementCount) / float64(totalCount)

				p.ThresholdsByRange[i].FFTThreshold = int(float64(existing.FFTThreshold)*existingWeight + float64(r.FFTThreshold)*newWeight)
				p.ThresholdsByRange[i].ParallelThreshold = int(float64(existing.ParallelThreshold)*existingWeight + float64(r.ParallelThreshold)*newWeight)
				p.ThresholdsByRange[i].ConfidenceScore = (existing.ConfidenceScore*existingWeight + r.ConfidenceScore*newWeight)
				p.ThresholdsByRange[i].MeasurementCount = totalCount
			}
			return
		}
	}

	// Add new range
	p.ThresholdsByRange = append(p.ThresholdsByRange, r)
}

// InitializeDefaultRanges initializes the profile with default range entries.
// This is useful when creating a new profile to ensure all ranges have some values.
func (p *CalibrationProfile) InitializeDefaultRanges() {
	if len(p.ThresholdsByRange) > 0 {
		return // Already has ranges
	}

	for _, r := range DefaultNRanges {
		p.ThresholdsByRange = append(p.ThresholdsByRange, RangeThresholds{
			MinN:              r.MinN,
			MaxN:              r.MaxN,
			FFTThreshold:      p.OptimalFFTThreshold,
			ParallelThreshold: p.OptimalParallelThreshold,
			StrassenThreshold: p.OptimalStrassenThreshold,
			ConfidenceScore:   0.3, // Low confidence for defaults
			MeasurementCount:  0,
		})
	}
}

// LoadOrCreate loads an existing profile or creates a new one if not found.
// If the existing profile is invalid for the current hardware, returns a new profile.
func LoadOrCreateProfile(path string) (*CalibrationProfile, bool) {
	profile, err := LoadProfile(path)
	if err != nil {
		// File doesn't exist or can't be read - create new
		return NewProfile(), false
	}

	if !profile.IsValid() {
		// Profile is incompatible with current hardware - create new
		return NewProfile(), false
	}

	return profile, true
}

// ProfileExists checks if a calibration profile exists at the given path.
func ProfileExists(path string) bool {
	if path == "" {
		path = GetDefaultProfilePath()
	}
	_, err := os.Stat(path)
	return err == nil
}
