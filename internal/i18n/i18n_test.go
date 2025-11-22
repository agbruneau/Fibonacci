package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDir(t *testing.T) {
	// Setup temporary directory for test files
	tmpDir := t.TempDir()

	// Create a valid translation file
	validContent := map[string]string{
		"TestKey": "TestValue",
		"ExecConfigTitle": "Overridden Title",
	}
	validJSON, _ := json.Marshal(validContent)
	if err := os.WriteFile(filepath.Join(tmpDir, "fr.json"), validJSON, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create an invalid JSON file
	if err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte("{invalid-json"), 0644); err != nil {
		t.Fatalf("Failed to create bad file: %v", err)
	}

	// Backup original Messages to restore after test
	originalMessages := make(map[string]string)
	for k, v := range Messages {
		originalMessages[k] = v
	}
	defer func() {
		Messages = originalMessages
	}()

	tests := []struct {
		name        string
		dir         string
		lang        string
		expectError bool
		verifyKey   string
		verifyValue string
	}{
		{
			name:        "Valid Load",
			dir:         tmpDir,
			lang:        "fr",
			expectError: false,
			verifyKey:   "TestKey",
			verifyValue: "TestValue",
		},
		{
			name:        "File Not Found",
			dir:         tmpDir,
			lang:        "es", // Doesn't exist
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			dir:         tmpDir,
			lang:        "bad",
			expectError: true,
		},
		{
			name:        "Empty Dir",
			dir:         "",
			lang:        "fr",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LoadFromDir(tt.dir, tt.lang)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if val, ok := Messages[tt.verifyKey]; !ok || val != tt.verifyValue {
					t.Errorf("Expected Messages[%s] = %s, got %s", tt.verifyKey, tt.verifyValue, val)
				}
				// Check that default keys are still there (or overridden)
				if Messages["ExecConfigTitle"] != "Overridden Title" {
					t.Errorf("Expected override of ExecConfigTitle")
				}
				if _, ok := Messages["CalibrationTitle"]; !ok {
					t.Error("Expected existing keys to remain")
				}
			}
		})
	}
}
