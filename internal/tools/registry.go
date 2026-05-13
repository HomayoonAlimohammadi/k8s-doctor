package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

type JSONSchema struct {
	Type       string                `json:"type"`
	Properties map[string]JSONSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Enum       []string              `json:"enum,omitempty"`
	Items      *JSONSchema           `json:"items,omitempty"`
}

type ToolResult struct {
	Summary  string          `json:"summary"`
	Data     map[string]any  `json:"data,omitempty"`
	Commands []CommandResult `json:"commands,omitempty"`
}

type Tool interface {
	Name() string
	Description() string
	InputSchema() JSONSchema
	Execute(ctx context.Context, input json.RawMessage) (ToolResult, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(tool Tool) error {
	if tool.Name() == "" {
		return fmt.Errorf("tool name is empty")
	}
	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name())
	}
	r.tools[tool.Name()] = tool
	return nil
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (ToolResult, error) {
	tool, ok := r.tools[name]
	if !ok {
		return ToolResult{}, fmt.Errorf("unknown tool %q", name)
	}
	return tool.Execute(ctx, input)
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
