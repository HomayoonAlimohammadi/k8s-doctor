package llm

import "context"

type FakeChatModel struct{ Response string }

func (f FakeChatModel) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return ChatResponse{Content: f.Response}, nil
}

type FakeEmbeddingModel struct{}

func (FakeEmbeddingModel) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, len(texts))
	for i, text := range texts {
		vectors[i] = []float64{float64(len(text)), 1}
	}
	return vectors, nil
}
