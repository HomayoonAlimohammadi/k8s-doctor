package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type echoTool struct{}

func (echoTool) Name() string            { return "echo" }
func (echoTool) Description() string     { return "echo test tool" }
func (echoTool) InputSchema() JSONSchema { return JSONSchema{Type: "object"} }
func (echoTool) Execute(ctx context.Context, input json.RawMessage) (ToolResult, error) {
	return ToolResult{Summary: string(input), Data: map[string]any{"ok": true}}, nil
}

func TestRegistryExecutesTool(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(echoTool{}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	result, err := r.Execute(context.Background(), "echo", []byte(`{"message":"hi"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Summary != `{"message":"hi"}` {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
}
