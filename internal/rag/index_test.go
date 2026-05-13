package rag

import (
	"context"
	"testing"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, len(texts))
	for i, text := range texts {
		if len(text) > 0 && (text[0] == 'd' || text[0] == 'D') {
			vectors[i] = []float64{1, 0}
		} else {
			vectors[i] = []float64{0, 1}
		}
	}
	return vectors, nil
}

func TestIndexSearchReturnsNearestChunk(t *testing.T) {
	idx := NewMemoryIndex(fakeEmbedder{})
	chunks := []Chunk{{ID: "dns", Text: "dns troubleshooting"}, {ID: "storage", Text: "storage troubleshooting"}}
	if err := idx.Add(context.Background(), chunks); err != nil {
		t.Fatalf("Add error: %v", err)
	}
	hits, err := idx.Search(context.Background(), "dns", 1)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(hits) != 1 || hits[0].Chunk.ID != "dns" {
		t.Fatalf("unexpected hits: %+v", hits)
	}
}
