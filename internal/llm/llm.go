package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []Message  `json:"messages"`
	Tools    []ToolSpec `json:"tools,omitempty"`
}

type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      any    `json:"schema"`
}

type ChatResponse struct {
	Content string `json:"content"`
}

type ChatModel interface {
	Complete(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type EmbeddingModel interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}
