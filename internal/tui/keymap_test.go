package tui_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"

	"github.com/agbruneau/FibGo/internal/tui"
)

func TestDefaultKeyMap_AllBindingsDefined(t *testing.T) {
	t.Parallel()
	km := tui.DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Quit", km.Quit},
		{"Pause", km.Pause},
		{"Reset", km.Reset},
		{"Up", km.Up},
		{"Down", km.Down},
		{"PageUp", km.PageUp},
		{"PageDown", km.PageDown},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()
			if !b.binding.Enabled() {
				t.Errorf("expected %s binding to be enabled", b.name)
			}
			keys := b.binding.Keys()
			if len(keys) == 0 {
				t.Errorf("expected %s binding to have at least one key", b.name)
			}
		})
	}
}

func TestDefaultKeyMap_QuitKeys(t *testing.T) {
	t.Parallel()
	km := tui.DefaultKeyMap()

	keys := km.Quit.Keys()
	hasQ := false
	hasCtrlC := false
	for _, k := range keys {
		switch k {
		case "q":
			hasQ = true
		case "ctrl+c":
			hasCtrlC = true
		}
	}

	if !hasQ {
		t.Error("expected Quit binding to include 'q'")
	}
	if !hasCtrlC {
		t.Error("expected Quit binding to include 'ctrl+c'")
	}
}
