package llm

import (
	"context"
	"log/slog"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type FakeChatModel struct{ Response string }

func (f FakeChatModel) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	slog.Warn("FakeChatModel.Complete called — returning canned response",
		"component", "llm.fake", "messages", len(req.Messages),
		"response_preview", logging.Truncate(f.Response, 200))
	return ChatResponse{Content: f.Response}, nil
}

type FakeEmbeddingModel struct{}

func (FakeEmbeddingModel) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	slog.Warn("FakeEmbeddingModel.Embed called — these vectors are NOT semantic",
		"component", "llm.fake", "texts", len(texts))
	vectors := make([][]float64, len(texts))
	for i, text := range texts {
		vectors[i] = []float64{float64(len(text)), 1}
	}
	return vectors, nil
}
