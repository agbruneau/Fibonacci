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
	"sync"
)

// messageStore holds the messages map and provides thread-safe access.
type messageStore struct {
	mu       sync.RWMutex
	messages map[string]string
}

// defaultMessages contains the built-in English messages.
var defaultMessages = map[string]string{
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

// store is the global message store instance.
var store = &messageStore{
	messages: make(map[string]string),
}

func init() {
	// Initialize with default messages
	for k, v := range defaultMessages {
		store.messages[k] = v
	}
}

// Messages provides backward-compatible access to messages.
// Note: Direct map access is not thread-safe. Use GetMessage() for concurrent access.
// This variable is kept for backward compatibility with existing code.
var Messages = store.messages

// GetMessage retrieves a message by key in a thread-safe manner.
// If the key is not found, it returns the key itself as a fallback,
// which helps identify missing translations during development.
//
// Parameters:
//   - key: The message key to look up.
//
// Returns:
//   - string: The message text, or the key if not found.
func GetMessage(key string) string {
	store.mu.RLock()
	defer store.mu.RUnlock()

	if msg, ok := store.messages[key]; ok {
		return msg
	}
	return key // Fallback to key if message not found
}

// SetMessage sets a message value in a thread-safe manner.
//
// Parameters:
//   - key: The message key.
//   - value: The message text.
func SetMessage(key, value string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.messages[key] = value
}

// LoadFromDir loads a JSON translation file from a given directory.
// On success, it replaces existing entries in Messages with those from the file,
// falling back on already present values. The expected format is a JSON object
// of the form { "Key": "Value", ... }.
//
// This function is thread-safe.
//
// Parameters:
//   - dir: The directory containing the translation files.
//   - lang: The language code (e.g., "en", "fr"). The function looks for a
//     file named `<lang>.json` in `dir`.
//
// Returns:
//   - error: An error if the directory or language is invalid, or if the file
//     cannot be read or parsed.
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

	// Merge: loaded entries replace default values (thread-safe)
	store.mu.Lock()
	defer store.mu.Unlock()
	for k, v := range loaded {
		store.messages[k] = v
	}
	return nil
}

// ResetToDefaults resets all messages to their default values.
// This is primarily useful for testing.
func ResetToDefaults() {
	store.mu.Lock()
	defer store.mu.Unlock()
	for k, v := range defaultMessages {
		store.messages[k] = v
	}
}
