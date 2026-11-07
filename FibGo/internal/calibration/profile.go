// This file implements calibration profile persistence.

package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/agbruneau/FibGo/internal/config"
)

// CalibrationProfile stores the results of a calibration run.
// It captures both the optimal thresholds and the hardware context
// to allow validation of cached results.
type CalibrationProfile struct {
	// Hardware identification
	CPUModel        string `json:"cpu_model"`
	NumCPU          int    `json:"num_cpu"`
	GOARCH          string `json:"goarch"`
	GOOS            string `json:"goos"`
	GoVersion       string `json:"go_version"`
	WordSize        int    `json:"word_size"`         // 32 or 64
	CPUHeuristicKey string `json:"cpu_heuristic_key"` // config.DetectHardwareHeuristic().HeuristicKey() — SIMD / arch class

	// Calibrated thresholds (default/fallback values)
	OptimalParallelThreshold int `json:"optimal_parallel_threshold"`
	OptimalFFTThreshold      int `json:"optimal_fft_threshold"`
	OptimalStrassenThreshold int `json:"optimal_strassen_threshold"`

	// Calibration metadata
	CalibratedAt    time.Time `json:"calibrated_at"`
	CalibrationN    uint64    `json:"calibration_n"`
	CalibrationTime string    `json:"calibration_time"`
	Confidence      float64   `json:"confidence"` // 0.0 to 1.0 indicating calibration reliability

	// Version for forward compatibility
	ProfileVersion int `json:"profile_version"`
}

const (
	// CurrentProfileVersion is the current version of the profile format.
	// Increment this when making breaking changes to the profile structure,
	// or when the MEANING of a stored value changes enough that replaying an
	// older profile would be worse than re-measuring.
	//
	// Version 3 added cpu_heuristic_key (SIMD / arch classification for default
	// thresholds). Version 4 is a semantic bump, not a structural one: audit
	// M-01 rewrote how optimal_fft_threshold is derived (sizes below bigfft's
	// own threshold no longer compared against themselves, a margin and a
	// monotonicity requirement, confidence earned rather than assumed), so a
	// version-3 profile can carry a value that the current code would never
	// produce. Invalidating them forces one re-calibration instead of pinning
	// a figure that was noise.
	CurrentProfileVersion = 4

	// DefaultProfileFileName is the default name for the calibration profile file.
	DefaultProfileFileName = ".fibcalc_calibration.json"
)

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
		CPUModel:        getCPUModel(),
		NumCPU:          runtime.NumCPU(),
		GOARCH:          runtime.GOARCH,
		GOOS:            runtime.GOOS,
		GoVersion:       runtime.Version(),
		WordSize:        32 << (^uint(0) >> 63), // 32 or 64
		CPUHeuristicKey: config.CurrentHardwareHeuristicKey(),
		CalibratedAt:    time.Now(),
		ProfileVersion:  CurrentProfileVersion,
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

// loadProfile loads a calibration profile from the specified path.
// Returns nil and an error if the file doesn't exist or can't be parsed.
func loadProfile(path string) (*CalibrationProfile, error) {
	if path == "" {
		path = GetDefaultProfilePath()
	}

	// Provenance of path, exhaustively — it reaches here by exactly four routes:
	//  1. the --calibration-profile flag (internal/config/config.go);
	//  2. FIBCALC_CALIBRATION_PROFILE, which applyEnvOverrides (internal/config/env.go)
	//     routes into the same cfg.CalibrationProfile whenever the flag was not set
	//     explicitly — so a wrapper script, CI runner or parent process can supply
	//     this path without it ever appearing on the command line;
	//  3. empty → GetDefaultProfilePath() with os.UserHomeDir() succeeding: an
	//     absolute path under HOME / USERPROFILE — also environment-derived;
	//  4. empty → GetDefaultProfilePath() with os.UserHomeDir() FAILING: it falls
	//     back to the bare relative DefaultProfileFileName, so the read resolves
	//     against the process working directory.
	// Routes 1-3 are inputs the operator already controls in a single-user CLI that
	// runs with the operator's own privileges: no privilege boundary exists for a
	// traversal to cross, and there is no confined root to escape.
	//
	// Route 4 is the one the operator does not necessarily control: with no home
	// directory resolvable, a process started in a directory someone else can write
	// reads that directory's .fibcalc_calibration.json. It is still not a traversal
	// — the name is a fixed constant, so no input selects the file — and the blast
	// radius is a JSON document unmarshalled into CalibrationProfile, then rejected
	// by IsValid unless it matches this machine's GOARCH, NumCPU, word size, profile
	// version and CPU heuristic key. A file that survives that only perturbs tuning
	// thresholds, and routes 1-2 override it for any caller that cares.
	//
	// gosec's os.Root remedy is declined deliberately: it would break the documented
	// ability to point the profile at an arbitrary path. SaveProfile writes that same
	// path by the same four routes, so read and write have identical exposure —
	// including route 4, where the write lands in the working directory.
	data, err := os.ReadFile(path) // #nosec G304 -- path is operator-controlled via flag, FIBCALC_CALIBRATION_PROFILE, or the HOME/USERPROFILE default; the home-lookup-failure fallback reads a fixed filename from the working directory (see above); single-user CLI with no privilege boundary
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
//
// The write is atomic: the JSON is first written to a temporary file in the
// SAME directory as the target, fsynced-equivalent via close, then renamed
// over the target. os.Rename is atomic within a volume on both POSIX and
// Windows, so a concurrent reader (or a crash mid-write) never observes a
// truncated or partially written profile — it sees either the old complete
// file or the new one. The temporary file is removed on any failure.
func (p *CalibrationProfile) SaveProfile(path string) error {
	if path == "" {
		path = GetDefaultProfilePath()
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp profile: %w", err)
	}
	tmpName := tmp.Name()

	// Ensure the temp file never lingers if anything below fails.
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp profile: %w", err)
	}

	// Match the historical 0600 permission (CreateTemp creates 0600 already,
	// but be explicit so the contract holds regardless of umask/platform).
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to chmod temp profile: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp profile: %w", err)
	}

	if err := renameAtomic(tmpName, path); err != nil {
		return fmt.Errorf("failed to finalize profile: %w", err)
	}

	cleanup = false
	return nil
}

// renameAtomic renames src over dst. The rename itself is atomic within a
// volume on both POSIX and Windows. On Windows, however, the replace fails
// with a sharing/access error if another process currently holds the target
// open (e.g. a concurrent reader), even though no truncation ever occurs.
// A bounded exponential backoff retry (delay = min(5ms << attempt, 50ms))
// absorbs that transient window so concurrent save/load across processes does
// not spuriously fail; on POSIX the first attempt succeeds and the loop is a
// no-op.
func renameAtomic(src, dst string) error {
	const (
		maxAttempts = 40
		baseDelay   = 5 * time.Millisecond
		maxDelay    = 50 * time.Millisecond
	)
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = os.Rename(src, dst); err == nil {
			return nil
		}
		if attempt < maxAttempts-1 {
			delay := baseDelay << attempt
			if delay > maxDelay || delay <= 0 {
				delay = maxDelay
			}
			time.Sleep(delay)
		}
	}
	return err
}

// IsValid checks if the profile is valid for the current hardware.
// A profile is considered valid if:
// - The profile version matches
// - The number of CPUs matches
// - The architecture matches
// - The word size matches
// - The CPU heuristic key (SIMD / arch class) matches
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

	if p.CPUHeuristicKey != config.CurrentHardwareHeuristicKey() {
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

	return fmt.Sprintf(
		"CalibrationProfile{CPU: %s, Parallel: %d bits, FFT: %d bits, Strassen: %d bits, Confidence: %.2f, Calibrated: %s}",
		p.CPUModel,
		p.OptimalParallelThreshold,
		p.OptimalFFTThreshold,
		p.OptimalStrassenThreshold,
		p.Confidence,
		p.CalibratedAt.Format(time.RFC3339),
	)
}

// LoadOrCreate loads an existing profile or creates a new one if not found.
// If the existing profile is invalid for the current hardware, returns a new profile.
func LoadOrCreateProfile(path string) (*CalibrationProfile, bool) {
	profile, err := loadProfile(path)
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
