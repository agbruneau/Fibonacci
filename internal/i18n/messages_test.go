// Package i18n provides tests for internationalization functionality.
package i18n

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGetMessage(t *testing.T) {
	// Reset to defaults before testing
	ResetToDefaults()

	t.Run("returns existing message", func(t *testing.T) {
		msg := GetMessage("ExecConfigTitle")
		expected := "--- Execution Configuration ---"
		if msg != expected {
			t.Errorf("expected %q, got %q", expected, msg)
		}
	})

	t.Run("returns key for missing message", func(t *testing.T) {
		msg := GetMessage("NonExistentKey")
		if msg != "NonExistentKey" {
			t.Errorf("expected key 'NonExistentKey' as fallback, got %q", msg)
		}
	})
}

func TestSetMessage(t *testing.T) {
	ResetToDefaults()

	t.Run("sets new message", func(t *testing.T) {
		SetMessage("TestKey", "Test Value")
		msg := GetMessage("TestKey")
		if msg != "Test Value" {
			t.Errorf("expected 'Test Value', got %q", msg)
		}
	})

	t.Run("overwrites existing message", func(t *testing.T) {
		original := GetMessage("ExecConfigTitle")
		SetMessage("ExecConfigTitle", "Custom Title")
		modified := GetMessage("ExecConfigTitle")
		if modified != "Custom Title" {
			t.Errorf("expected 'Custom Title', got %q", modified)
		}

		// Restore original
		SetMessage("ExecConfigTitle", original)
	})
}

func TestResetToDefaults(t *testing.T) {
	// Modify a default message
	SetMessage("ExecConfigTitle", "Modified Value")

	// Reset
	ResetToDefaults()

	// Check default is restored
	msg := GetMessage("ExecConfigTitle")
	expected := "--- Execution Configuration ---"
	if msg != expected {
		t.Errorf("expected default %q, got %q", expected, msg)
	}
}

func TestLoadFromDir(t *testing.T) {
	ResetToDefaults()

	t.Run("returns error for empty directory", func(t *testing.T) {
		err := LoadFromDir("", "en")
		if err == nil {
			t.Error("expected error for empty directory")
		}
	})

	t.Run("returns error for empty language", func(t *testing.T) {
		err := LoadFromDir("./locales", "")
		if err == nil {
			t.Error("expected error for empty language")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		err := LoadFromDir("./nonexistent", "en")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("loads valid JSON file", func(t *testing.T) {
		// Create a temporary directory with a test locale file
		tmpDir := t.TempDir()
		testContent := `{"TestMessage": "Bonjour", "ExecConfigTitle": "Configuration d'exécution"}`
		err := os.WriteFile(filepath.Join(tmpDir, "fr.json"), []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err = LoadFromDir(tmpDir, "fr")
		if err != nil {
			t.Fatalf("LoadFromDir failed: %v", err)
		}

		// Check loaded message
		msg := GetMessage("TestMessage")
		if msg != "Bonjour" {
			t.Errorf("expected 'Bonjour', got %q", msg)
		}

		// Check overwritten message
		configMsg := GetMessage("ExecConfigTitle")
		if configMsg != "Configuration d'exécution" {
			t.Errorf("expected 'Configuration d'exécution', got %q", configMsg)
		}

		// Reset for other tests
		ResetToDefaults()
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidJSON := `{"invalid": json}`
		err := os.WriteFile(filepath.Join(tmpDir, "bad.json"), []byte(invalidJSON), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err = LoadFromDir(tmpDir, "bad")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestDefaultMessages(t *testing.T) {
	ResetToDefaults()

	// Verify all expected default messages exist
	expectedKeys := []string{
		"CalibrationTitle",
		"CalibrationSummary",
		"OptimalRecommendation",
		"ExecConfigTitle",
		"ExecStartTitle",
		"ComparisonSummary",
		"GlobalStatusSuccess",
		"GlobalStatusFailure",
		"StatusCriticalMismatch",
		"StatusCanceled",
		"StatusTimeout",
		"StatusFailure",
	}

	for _, key := range expectedKeys {
		msg := GetMessage(key)
		if msg == key {
			t.Errorf("default message missing for key %q", key)
		}
	}
}

func TestMessagesMapBackwardCompatibility(t *testing.T) {
	ResetToDefaults()

	// The Messages variable should provide access to the same messages
	// Note: Direct map access is not thread-safe, this is for backward compatibility
	if Messages == nil {
		t.Fatal("Messages map should not be nil")
	}

	msg, ok := Messages["ExecConfigTitle"]
	if !ok {
		t.Error("Messages map should contain 'ExecConfigTitle'")
	}
	if msg != "--- Execution Configuration ---" {
		t.Errorf("unexpected value in Messages map: %q", msg)
	}
}

func TestConcurrentAccess(t *testing.T) {
	ResetToDefaults()

	// Test that concurrent read/write operations don't cause race conditions
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = GetMessage("ExecConfigTitle")
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				SetMessage("ConcurrentKey", "value")
			}
		}(i)
	}

	wg.Wait()

	// If we get here without a race condition panic, the test passes
	ResetToDefaults()
}
