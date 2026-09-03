package completion

// FlagCompletion describes a CLI flag for shell completion generation.
// All shell completion functions generate from this registry, so adding
// a new flag only requires appending to flagRegistry. TestFlagRegistryInSyncWithConfig
// enforces that this list stays in sync with config.registerFlags (the canonical
// flag set), so a new CLI flag added there fails the build until it is added here.
type FlagCompletion struct {
	Long      string   // long flag name without "--" (e.g., "help")
	Short     string   // short flag without "-" (e.g., "h")
	Help      string   // description text
	Values    []string // suggested completion values (nil = boolean/no suggestions)
	ValueName string   // label for the value in zsh (e.g., "number", "duration")
	IsFile    bool     // true if the flag takes a file path
	IsAlgo    bool     // true if values come from algorithm list (dynamic)
	BashGroup string   // flags with same non-empty BashGroup share a bash case entry
}

// flagRegistry is the central list of all CLI flags for completion generation.
// The order matches the original completion output for each shell.
var flagRegistry = []FlagCompletion{
	{Long: "help", Short: "h", Help: "Show help message"},
	{Long: "version", Short: "V", Help: "Show version information"},
	{Short: "n", Help: "Fibonacci index to calculate", ValueName: "number"},
	{Long: "verbose", Short: "v", Help: "Display the full result value (can be very long)"},
	{Long: "details", Short: "d", Help: "Show performance details"},
	{Long: "timeout", Help: "Maximum execution time", Values: []string{"1m", "5m", "10m", "30m", "1h"}, ValueName: "duration"},
	{Long: "algo", Help: "Algorithm to use", IsAlgo: true, ValueName: "algorithm"},
	// "-1" disables the threshold outright (config.ThresholdDisabled); it is a
	// valid value since audit H-02 and is what calibration persists when the
	// sequential / no-FFT baseline wins, so it belongs in the suggestions.
	{Long: "threshold", Help: "Parallelism threshold in bits (-1 disables)", Values: []string{"-1", "1024", "2048", "4096", "8192", "16384"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "fft-threshold", Help: "FFT threshold in bits (-1 disables)", Values: []string{"-1", "100000", "500000", "1000000"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "strassen-threshold", Help: "Strassen threshold", Values: []string{"1024", "2048", "3072", "4096"}, ValueName: "bits", BashGroup: "threshold"},
	{Long: "calibrate", Help: "Run calibration mode"},
	{Long: "auto-calibrate", Help: "Enable auto-calibration"},
	{Long: "calibration-profile", Help: "Calibration profile file", IsFile: true, ValueName: "file"},
	{Long: "output", Short: "o", Help: "Output file path", IsFile: true, ValueName: "file"},
	{Long: "quiet", Short: "q", Help: "Quiet mode for scripts"},
	{Long: "machine", Help: "Machine-readable output (no ANSI colors)"},
	{Long: "completion", Help: "Generate completion script", Values: []string{"bash", "zsh", "fish", "powershell"}, ValueName: "shell"},
	{Long: "calculate", Short: "c", Help: "Display the calculated value (disabled by default)"},
	{Long: "tui", Help: "Launch interactive TUI dashboard"},
	{Long: "last-digits", Help: "Compute only the last K decimal digits", ValueName: "count"},
	{Long: "memory-limit", Help: "Maximum memory budget (e.g., 8G, 512M)", ValueName: "size"},
	{Long: "gc-control", Help: "GC control during calculation", Values: []string{"auto", "aggressive", "disabled"}, ValueName: "mode"},
	{Long: "dynamic-thresholds", Help: "Adjust FFT/parallelism thresholds during the calculation"},
}

// flagKey returns the identifier used for lookups: Long name if present, else Short.
func flagKey(f FlagCompletion) string {
	if f.Long != "" {
		return f.Long
	}
	return f.Short
}
