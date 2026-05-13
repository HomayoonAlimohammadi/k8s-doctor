package audit

import (
	"context"
	"encoding/json"
	"fmt"
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
	return &Logger{path: path}
}

func (l *Logger) Record(ctx context.Context, entry Entry) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}
	return nil
}
