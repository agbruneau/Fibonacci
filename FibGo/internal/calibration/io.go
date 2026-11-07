// This file holds what is left of calibration's own output helpers after the
// presentation port (audit ARC-01): one line of narration, phrased here and
// rendered by the adapter. printCalibrationResults moved to internal/cli as the
// CalibrationReporter.Summary implementation, taking the tabwriter, the colors
// and the internal/format dependency with it.

package calibration

import "github.com/agbruneau/FibGo/internal/config"

// printCalibrationOutput reports the thresholds a calibration settled on.
//
// Parameters:
//   - cfg: The updated configuration with calibration results.
//   - rep: The reporter that renders it.
func printCalibrationOutput(cfg config.AppConfig, rep Reporter) {
	rep.Notice("Auto-calibration: parallelism=%d bits, FFT=%d bits, Strassen=%d bits",
		cfg.Threshold, cfg.FFTThreshold, cfg.StrassenThreshold)
}
