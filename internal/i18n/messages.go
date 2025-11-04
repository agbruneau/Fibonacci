// Package i18n centralizes user-facing messages for the CLI.
// It provides a simple basis for internationalization and ensures
// uniformity in the tone and vocabulary displayed by the application.
package i18n

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
)

// Messages groups user-facing messages (basic i18n).
// Centralizing these labels facilitates maintenance, consistency, and a
// potential multi-language translation in the future.
var Messages = map[string]string{
    "CalibrationTitle":       "--- Calibration Mode: Finding the Optimal Parallelism Threshold ---",
    "CalibrationSummary":     "--- Calibration Summary ---",
    "OptimalRecommendation":  "✅ Recommendation for this machine: --threshold %d",
    "ExecConfigTitle":        "--- Execution Configuration ---",
    "ExecStartTitle":         "--- Starting Execution ---",
    "ComparisonSummary":      "--- Comparison Summary ---",
    "GlobalStatusSuccess":    "Global Status: Success. All valid results are consistent.",
    "GlobalStatusFailure":    "Global Status: Failure. No algorithm could complete the calculation.",
    "StatusCriticalMismatch": "Global Status: CRITICAL ERROR! An inconsistency was detected between the results of the algorithms.",
    "StatusCanceled":         "Status: Canceled",
    "StatusTimeout":          "Status: Failure (Timeout). The execution limit of %s was reached%s.",
    "StatusFailure":          "Status: Failure. An unexpected error occurred: %v",
}


// LoadFromDir loads a JSON translation file (e.g., en.json) from
// a given directory. On success, it replaces existing entries in
// Messages with those from the file (falling back on already present values).
// The expected format is a JSON object { "Key": "Value", ... }.
func LoadFromDir(dir string, lang string) error {
    if dir == "" || lang == "" {
        return errors.New("i18n: empty directory or language")
    }
    path := filepath.Join(dir, fmt.Sprintf("%s.json", lang))
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()
    dec := json.NewDecoder(f)
    loaded := map[string]string{}
    if err := dec.Decode(&loaded); err != nil {
        return err
    }
    // Merge: loaded entries replace default values
    for k, v := range loaded {
        Messages[k] = v
    }
    return nil
}
