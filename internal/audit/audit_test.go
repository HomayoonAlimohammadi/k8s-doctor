package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoggerWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	logger := NewLogger(filepath.Join(dir, "audit.jsonl"))

	entry := Entry{SessionID: "s1", Tool: "cluster_status", Input: map[string]any{"node": "cp1"}, Result: "ok"}
	if err := logger.Record(context.Background(), entry); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var got Entry
	if err := json.Unmarshal(raw[:len(raw)-1], &got); err != nil {
		t.Fatalf("audit line is not JSON: %v", err)
	}
	if got.SessionID != "s1" || got.Tool != "cluster_status" || got.Result != "ok" || got.Timestamp.IsZero() {
		t.Fatalf("unexpected entry: %+v", got)
	}
}
