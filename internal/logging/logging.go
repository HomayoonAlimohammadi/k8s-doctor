// Package logging configures the process-wide slog logger and exposes a few
// helpers used by the rest of the codebase. We intentionally lean on
// slog.Default() everywhere rather than threading loggers through every
// constructor — this is a prototype, and a single global keeps the wiring
// readable.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

// Setup builds a slog.Logger from the given level/format strings and installs
// it as the process default. Unknown values fall back to the auto format and
// debug level. It also returns the logger so callers can attach component
// attributes if they want.
//
// Format values:
//   - "" or "auto": tint (colored) when out is a TTY and color is not
//     disabled via NO_COLOR / TERM=dumb, else plain text
//   - "color" or "tint": always colored
//   - "text": plain text, never colored
//   - "json": JSON lines
func Setup(level, format string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stderr
	}

	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug", "":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelDebug
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" || format == "auto" {
		if colorEnabled(out) {
			format = "color"
		} else {
			format = "text"
		}
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: lvl, AddSource: true})
	case "color", "tint":
		handler = tint.NewHandler(out, &tint.Options{
			Level:      lvl,
			AddSource:  true,
			TimeFormat: time.Kitchen,
			NoColor:    false,
		})
	default:
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{Level: lvl, AddSource: true})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// colorEnabled reports whether ANSI colors should be used for the given
// writer. It respects the standard NO_COLOR convention (https://no-color.org)
// and treats TERM=dumb as monochrome. Otherwise it enables color only when
// the writer is a terminal — determined by checking for an *os.File backed
// by stdout/stderr with a TTY-ish environment.
func colorEnabled(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if term := os.Getenv("TERM"); term == "dumb" {
		return false
	}
	// FORCE_COLOR=1 short-circuits the TTY check (useful in CI that wraps
	// output but still understands ANSI).
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	// Cheap, dep-free TTY heuristic: stdout/stderr to a real terminal will
	// have a non-zero size in stat (named device) and no offset support.
	// We use os.ModeCharDevice from FileMode which is set for character
	// devices like a TTY.
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Truncate returns s shortened to at most n runes, appending an ellipsis when
// the value was cut. It is safe on empty strings and non-positive n.
func Truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "...<truncated>"
}

// RedactBearer returns a placeholder string when key is non-empty, suitable
// for logging without leaking secrets.
func RedactBearer(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return "<redacted>"
	}
	return key[:4] + "...<redacted>"
}
