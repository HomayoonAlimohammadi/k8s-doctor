package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (c OpenAIClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	endpoint := c.BaseURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	slog.Debug("openai chat begin",
		"component", "llm.openai", "endpoint", endpoint, "model", c.Model,
		"messages", len(req.Messages), "tools", len(req.Tools),
		"api_key", logging.RedactBearer(c.APIKey))

	body := map[string]any{"model": c.Model, "messages": req.Messages}
	raw, err := json.Marshal(body)
	if err != nil {
		slog.Error("openai marshal request failed", "component", "llm.openai", "error", err)
		return ChatResponse{}, fmt.Errorf("marshal openai request: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		slog.Error("openai create request failed", "component", "llm.openai", "error", err)
		return ChatResponse{}, fmt.Errorf("create openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		slog.Error("openai http call failed", "component", "llm.openai",
			"endpoint", endpoint, "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return ChatResponse{}, fmt.Errorf("call openai-compatible API: %w", err)
	}
	defer resp.Body.Close()
	slog.Debug("openai http response",
		"component", "llm.openai", "status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("openai non-2xx status",
			"component", "llm.openai", "status", resp.StatusCode,
			"body", logging.Truncate(string(body), 1024))
		return ChatResponse{}, fmt.Errorf("openai-compatible API status %d", resp.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		slog.Error("openai decode response failed", "component", "llm.openai", "error", err)
		return ChatResponse{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		slog.Error("openai response empty", "component", "llm.openai")
		return ChatResponse{}, fmt.Errorf("openai response contained no choices")
	}
	content := decoded.Choices[0].Message.Content
	slog.Debug("openai chat complete",
		"component", "llm.openai", "model", c.Model,
		"chars", len(content), "preview", logging.Truncate(content, 200))
	return ChatResponse{Content: content}, nil
}

// OpenAIEmbedder implements rag.Embedder against an OpenAI-compatible
// /v1/embeddings endpoint.
type OpenAIEmbedder struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (e OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	endpoint := e.BaseURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/embeddings"
	}
	slog.Debug("openai embed begin",
		"component", "llm.openai.embed", "endpoint", endpoint, "model", e.Model,
		"texts", len(texts), "api_key", logging.RedactBearer(e.APIKey))

	body := map[string]any{"model": e.Model, "input": texts}
	raw, err := json.Marshal(body)
	if err != nil {
		slog.Error("openai embed marshal failed", "component", "llm.openai.embed", "error", err)
		return nil, fmt.Errorf("marshal openai embed request: %w", err)
	}
	httpClient := e.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		slog.Error("openai embed create request failed", "component", "llm.openai.embed", "error", err)
		return nil, fmt.Errorf("create openai embed request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if e.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.APIKey)
	}

	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		slog.Error("openai embed http call failed", "component", "llm.openai.embed",
			"endpoint", endpoint, "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return nil, fmt.Errorf("call openai embed API: %w", err)
	}
	defer resp.Body.Close()
	slog.Debug("openai embed http response",
		"component", "llm.openai.embed", "status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("openai embed non-2xx status",
			"component", "llm.openai.embed", "status", resp.StatusCode,
			"body", logging.Truncate(string(body), 1024))
		return nil, fmt.Errorf("openai embed API status %d", resp.StatusCode)
	}
	var decoded struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		slog.Error("openai embed decode failed", "component", "llm.openai.embed", "error", err)
		return nil, fmt.Errorf("decode openai embed response: %w", err)
	}
	if len(decoded.Data) != len(texts) {
		slog.Warn("openai embed returned unexpected count",
			"component", "llm.openai.embed", "got", len(decoded.Data), "want", len(texts))
	}
	vectors := make([][]float64, len(decoded.Data))
	for i := range decoded.Data {
		vectors[i] = decoded.Data[i].Embedding
	}
	slog.Debug("openai embed complete",
		"component", "llm.openai.embed", "vectors", len(vectors))
	return vectors, nil
}
