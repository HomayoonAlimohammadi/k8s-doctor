package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp  time.Time      `json:"timestamp"`
	SessionID  string         `json:"session_id"`
	Tool       string         `json:"tool"`
	Input      map[string]any `json:"input,omitempty"`
	Commands   []Command      `json:"commands,omitempty"`
	Result     string         `json:"result"`
	Error      string         `json:"error,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`
}

type Command struct {
	Args       []string `json:"args"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
}

type Logger struct {
	path string
	mu   sync.Mutex
}

func NewLogger(path string) *Logger {
	slog.Debug("audit.NewLogger", "component", "audit", "path", path)
	return &Logger{path: path}
}

func (l *Logger) Record(ctx context.Context, entry Entry) error {
	select {
	case <-ctx.Done():
		slog.Warn("audit.Record cancelled",
			"component", "audit", "tool", entry.Tool, "error", ctx.Err())
		return ctx.Err()
	default:
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		slog.Error("audit.Record mkdir failed",
			"component", "audit", "path", l.path, "error", err)
		return fmt.Errorf("create audit directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Error("audit.Record open failed",
			"component", "audit", "path", l.path, "error", err)
		return fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(entry); err != nil {
		slog.Error("audit.Record encode failed",
			"component", "audit", "path", l.path, "tool", entry.Tool, "error", err)
		return fmt.Errorf("write audit entry: %w", err)
	}
	slog.Debug("audit.Record ok",
		"component", "audit", "path", l.path,
		"session_id", entry.SessionID, "tool", entry.Tool,
		"commands", len(entry.Commands), "duration_ms", entry.DurationMS)
	return nil
}
