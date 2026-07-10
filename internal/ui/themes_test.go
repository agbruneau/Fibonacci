package ui_test

import (
	"os"
	"sync"
	"testing"

	"github.com/agbruneau/FibGo/internal/ui"
)

// stateMu serializes the tests below against each other: InitTheme,
// SetCurrentTheme and GetCurrentTUITheme all read/write ui's single
// process-wide active theme, and some of them also mutate the NO_COLOR /
// FIBCALC_TUI_THEME environment variables. Each test still calls
// t.Parallel() so it runs alongside other packages' suites; the mutex only
// keeps this package's own state-mutating tests from interleaving with each
// other.
//
// ponytail: a per-test mutex is the smallest fix for real t.Parallel()
// coverage over a package-level singleton without weakening any assertion.
// Threading the theme through a context instead of a global would remove
// the need for it, but that is a production API change outside this pass.
var stateMu sync.Mutex

// withCleanState locks stateMu, snapshots the current theme and the two
// environment variables these tests touch, and returns a func that restores
// them and unlocks. Not t.Setenv: callers here run under t.Parallel(), and
// Setenv forbids parallel ancestors.
func withCleanState(t *testing.T) func() {
	t.Helper()
	stateMu.Lock()
	theme := ui.GetCurrentTheme()
	noColor, hadNoColor := os.LookupEnv("NO_COLOR")
	tuiTheme, hadTUITheme := os.LookupEnv("FIBCALC_TUI_THEME")
	return func() {
		ui.SetCurrentTheme(theme)
		restoreEnv("NO_COLOR", noColor, hadNoColor)
		restoreEnv("FIBCALC_TUI_THEME", tuiTheme, hadTUITheme)
		stateMu.Unlock()
	}
}

func restoreEnv(key, value string, had bool) {
	if had {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}

// TestThemeLiterals verifies the two CLI theme literals never touch the
// active-theme singleton, so it never needs stateMu.
func TestThemeLiterals(t *testing.T) {
	t.Parallel()

	fields := func(th ui.Theme) map[string]string {
		return map[string]string{
			"Primary": th.Primary, "Secondary": th.Secondary, "Success": th.Success,
			"Warning": th.Warning, "Error": th.Error, "Info": th.Info,
			"Bold": th.Bold, "Underline": th.Underline, "Reset": th.Reset,
		}
	}

	t.Run("DarkTheme colors are all set", func(t *testing.T) {
		t.Parallel()
		for name, v := range fields(ui.DarkTheme) {
			if v == "" {
				t.Errorf("DarkTheme.%s is empty", name)
			}
		}
	})

	t.Run("NoColorTheme colors are all empty", func(t *testing.T) {
		t.Parallel()
		for name, v := range fields(ui.NoColorTheme) {
			if v != "" {
				t.Errorf("NoColorTheme.%s = %q, want empty", name, v)
			}
		}
	})
}

func TestInitTheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		noColor  bool
		envSet   bool
		envValue string
		want     string
	}{
		{name: "flag disables colors", noColor: true, want: "none"},
		{name: "flag false with no env uses dark theme", noColor: false, want: "dark"},
		{name: "NO_COLOR=1 disables colors", envSet: true, envValue: "1", want: "none"},
		{name: "NO_COLOR empty value still disables colors", envSet: true, envValue: "", want: "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := withCleanState(t)
			defer restore()

			if tt.envSet {
				os.Setenv("NO_COLOR", tt.envValue)
			} else {
				os.Unsetenv("NO_COLOR")
			}

			ui.InitTheme(tt.noColor)

			if got := ui.GetCurrentTheme().Name; got != tt.want {
				t.Errorf("InitTheme(%v) with NO_COLOR set=%v value=%q: theme = %q, want %q",
					tt.noColor, tt.envSet, tt.envValue, got, tt.want)
			}
		})
	}
}

// colorFuncs pairs each CLI color helper with the Theme field it must
// mirror, so TestColorFunctions can drive both over one table.
var colorFuncs = []struct {
	name  string
	fn    func() string
	field func(ui.Theme) string
}{
	{"ColorReset", ui.ColorReset, func(th ui.Theme) string { return th.Reset }},
	{"ColorRed", ui.ColorRed, func(th ui.Theme) string { return th.Error }},
	{"ColorGreen", ui.ColorGreen, func(th ui.Theme) string { return th.Success }},
	{"ColorYellow", ui.ColorYellow, func(th ui.Theme) string { return th.Warning }},
	{"ColorBlue", ui.ColorBlue, func(th ui.Theme) string { return th.Primary }},
	{"ColorMagenta", ui.ColorMagenta, func(th ui.Theme) string { return th.Info }},
	{"ColorCyan", ui.ColorCyan, func(th ui.Theme) string { return th.Secondary }},
	{"ColorBold", ui.ColorBold, func(th ui.Theme) string { return th.Bold }},
	{"ColorUnderline", ui.ColorUnderline, func(th ui.Theme) string { return th.Underline }},
}

func TestColorFunctions(t *testing.T) {
	t.Parallel()

	for _, theme := range []ui.Theme{ui.DarkTheme, ui.NoColorTheme} {
		t.Run(theme.Name, func(t *testing.T) {
			restore := withCleanState(t)
			defer restore()

			ui.SetCurrentTheme(theme)
			for _, cf := range colorFuncs {
				want := cf.field(theme)
				if got := cf.fn(); got != want {
					t.Errorf("%s() = %q, want %q", cf.name, got, want)
				}
			}
		})
	}
}

func TestGetCurrentTUITheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		theme    ui.Theme
		envSet   bool
		envValue string
		want     ui.TUITheme
	}{
		{name: "no-color theme ignores env", theme: ui.NoColorTheme, want: ui.NoColorTUITheme},
		{name: "dark theme with no env", theme: ui.DarkTheme, want: ui.DarkTUITheme},
		{name: "dark theme with high-contrast env", theme: ui.DarkTheme, envSet: true, envValue: "high-contrast", want: ui.HighContrastTUITheme},
		{name: "dark theme with highcontrast env (no dash)", theme: ui.DarkTheme, envSet: true, envValue: "highcontrast", want: ui.HighContrastTUITheme},
		{name: "dark theme with unrelated env value", theme: ui.DarkTheme, envSet: true, envValue: "solarized", want: ui.DarkTUITheme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := withCleanState(t)
			defer restore()

			ui.SetCurrentTheme(tt.theme)
			if tt.envSet {
				os.Setenv("FIBCALC_TUI_THEME", tt.envValue)
			} else {
				os.Unsetenv("FIBCALC_TUI_THEME")
			}

			if got := ui.GetCurrentTUITheme(); got != tt.want {
				t.Errorf("GetCurrentTUITheme() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
