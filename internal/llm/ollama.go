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

type OllamaClient struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func (c OllamaClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	endpoint := c.BaseURL
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/generate"
	}
	prompt := ""
	for _, msg := range req.Messages {
		prompt += msg.Role + ": " + msg.Content + "\n"
	}
	slog.Debug("ollama chat begin",
		"component", "llm.ollama", "endpoint", endpoint, "model", c.Model,
		"messages", len(req.Messages), "prompt_chars", len(prompt))

	body := map[string]any{"model": c.Model, "prompt": prompt, "stream": false}
	raw, err := json.Marshal(body)
	if err != nil {
		slog.Error("ollama marshal request failed", "component", "llm.ollama", "error", err)
		return ChatResponse{}, fmt.Errorf("marshal ollama request: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		slog.Error("ollama create request failed", "component", "llm.ollama", "error", err)
		return ChatResponse{}, fmt.Errorf("create ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		slog.Error("ollama http call failed", "component", "llm.ollama",
			"endpoint", endpoint, "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return ChatResponse{}, fmt.Errorf("call ollama API: %w", err)
	}
	defer resp.Body.Close()
	slog.Debug("ollama http response",
		"component", "llm.ollama", "status", resp.StatusCode,
		"duration_ms", time.Since(start).Milliseconds())

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("ollama non-2xx status",
			"component", "llm.ollama", "status", resp.StatusCode,
			"body", logging.Truncate(string(body), 1024))
		return ChatResponse{}, fmt.Errorf("ollama API status %d", resp.StatusCode)
	}
	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		slog.Error("ollama decode response failed", "component", "llm.ollama", "error", err)
		return ChatResponse{}, fmt.Errorf("decode ollama response: %w", err)
	}
	slog.Debug("ollama chat complete",
		"component", "llm.ollama", "model", c.Model,
		"chars", len(decoded.Response), "preview", logging.Truncate(decoded.Response, 200))
	return ChatResponse{Content: decoded.Response}, nil
}

// OllamaEmbedder implements rag.Embedder against the Ollama /api/embeddings
// endpoint. Ollama returns one vector per request, so we issue N calls.
type OllamaEmbedder struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func (e OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	endpoint := e.BaseURL
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/embeddings"
	}
	slog.Debug("ollama embed begin",
		"component", "llm.ollama.embed", "endpoint", endpoint, "model", e.Model,
		"texts", len(texts))

	httpClient := e.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	vectors := make([][]float64, 0, len(texts))
	for idx, text := range texts {
		body := map[string]any{"model": e.Model, "prompt": text}
		raw, err := json.Marshal(body)
		if err != nil {
			slog.Error("ollama embed marshal failed",
				"component", "llm.ollama.embed", "index", idx, "error", err)
			return nil, fmt.Errorf("marshal ollama embed request: %w", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
		if err != nil {
			slog.Error("ollama embed create request failed",
				"component", "llm.ollama.embed", "index", idx, "error", err)
			return nil, fmt.Errorf("create ollama embed request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			slog.Error("ollama embed http call failed",
				"component", "llm.ollama.embed", "index", idx,
				"duration_ms", time.Since(start).Milliseconds(), "error", err)
			return nil, fmt.Errorf("call ollama embed API: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			slog.Error("ollama embed non-2xx status",
				"component", "llm.ollama.embed", "index", idx, "status", resp.StatusCode,
				"body", logging.Truncate(string(body), 1024))
			return nil, fmt.Errorf("ollama embed API status %d", resp.StatusCode)
		}

		var decoded struct {
			Embedding []float64 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			resp.Body.Close()
			slog.Error("ollama embed decode failed",
				"component", "llm.ollama.embed", "index", idx, "error", err)
			return nil, fmt.Errorf("decode ollama embed response: %w", err)
		}
		resp.Body.Close()
		slog.Debug("ollama embed item",
			"component", "llm.ollama.embed", "index", idx,
			"duration_ms", time.Since(start).Milliseconds(), "dims", len(decoded.Embedding))
		vectors = append(vectors, decoded.Embedding)
	}
	slog.Debug("ollama embed complete", "component", "llm.ollama.embed", "vectors", len(vectors))
	return vectors, nil
}
