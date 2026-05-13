package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OllamaClient struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func (c OllamaClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	prompt := ""
	for _, msg := range req.Messages {
		prompt += msg.Role + ": " + msg.Content + "\n"
	}
	body := map[string]any{"model": c.Model, "prompt": prompt, "stream": false}
	raw, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal ollama request: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	endpoint := c.BaseURL
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/generate"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call ollama API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf("ollama API status %d", resp.StatusCode)
	}
	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("decode ollama response: %w", err)
	}
	return ChatResponse{Content: decoded.Response}, nil
}
