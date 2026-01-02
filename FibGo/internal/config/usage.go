package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/agbru/fibcalc/internal/ui"
)

// setCustomUsage configures the flag set with a colored usage function.
func setCustomUsage(fs *flag.FlagSet) {
	fs.Usage = func() {
		// Respect NO_COLOR even before app initialization
		t := ui.GetCurrentTheme()
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			t = ui.NoColorTheme
		}

		out := fs.Output()

		// Header
		fmt.Fprintf(out, "\n%sFibonacci Calculator%s\n", t.Bold, t.Reset)
		fmt.Fprintf(out, "High-performance modular Fibonacci calculator.\n\n")
		fmt.Fprintf(out, "%sUsage:%s\n  %s [flags]\n\n%sFlags:%s\n", t.Warning, t.Reset, fs.Name(), t.Warning, t.Reset)

		fs.VisitAll(func(f *flag.Flag) {
			name, usage := flag.UnquoteUsage(f)
			flagSig := fmt.Sprintf("-%s", f.Name)
			if len(name) > 0 {
				flagSig += " " + name
			}

			// Print formatted flag
			fmt.Fprintf(out, "  %s%-25s%s %s", t.Primary, flagSig, t.Reset, usage)

			// Print default value if meaningful
			if f.DefValue != "" && f.DefValue != "0" && f.DefValue != "false" {
				fmt.Fprintf(out, " %s(default %s)%s", t.Secondary, f.DefValue, t.Reset)
			}
			fmt.Fprintln(out)
		})
		fmt.Fprintln(out)
	}
}
