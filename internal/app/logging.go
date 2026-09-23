package app

import (
	"log/slog"
	"strings"
)

// parseLogLevel maps the --log-level value to a slog level.
//
// The second result reports whether logging is enabled at all: "off" (and the
// empty string, which is what an AppConfig built programmatically carries)
// means no logger, not a logger set to a high threshold. The distinction
// matters because a nil logger lets every consumer skip attribute construction
// entirely.
func parseLogLevel(v string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		// "off", "" and anything config.Validate would already have rejected.
		return slog.LevelInfo, false
	}
}

// newDiagnosticLogger builds the process logger from --log-level (or
// FIBCALC_LOG_LEVEL), returning nil when logging is off.
//
// This is the composition point the audit (OBS-01 / API-03) found missing. Four
// useful records already existed in the domain packages — GC disabled and
// re-enabled with heap sizes and cycle counts, dynamic threshold adjustments,
// FFT transform-cache hit rates, and a per-calculation summary — but each was
// wired to a discarding logger reachable only from a test-only setter, or
// filtered out by a process-wide level that Run itself set to Info. So the
// domain depended on a third-party logging library, which the book warns
// against (ch. 14, "keep it confined to the adapter layer"), and produced
// nothing in exchange.
//
// Decisions:
//
//   - stderr, never stdout. stdout carries results, which scripts parse
//     (ERR-02); a diagnostic stream interleaved with them would be a
//     regression, not an improvement.
//   - Off by default. These are debug records. A user who did not ask for them
//     gets nil, which every consumer turns into slog.DiscardHandler.
//   - JSON under --machine, text otherwise: --machine already means "output for
//     a program", and the same reasoning applies to this stream.
//   - No timestamp under --machine: machine output is meant to be diffable
//     between runs, and a clock reading makes every run differ.
func (a *Application) newDiagnosticLogger() *slog.Logger {
	level, enabled := parseLogLevel(a.Config.LogLevel)
	if !enabled {
		return nil
	}

	opts := &slog.HandlerOptions{Level: level}
	if !a.Config.MachineOutput {
		return slog.New(slog.NewTextHandler(a.ErrWriter, opts))
	}

	opts.ReplaceAttr = func(_ []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return attr
	}
	return slog.New(slog.NewJSONHandler(a.ErrWriter, opts))
}
