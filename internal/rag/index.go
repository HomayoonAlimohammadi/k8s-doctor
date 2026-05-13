package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
)

type SearchHit struct {
	Chunk Chunk
	Score float64
}

type MemoryIndex struct {
	embedder Embedder
	chunks   []Chunk
}

func NewMemoryIndex(embedder Embedder) *MemoryIndex {
	return &MemoryIndex{embedder: embedder}
}

func (i *MemoryIndex) Add(ctx context.Context, chunks []Chunk) error {
	texts := make([]string, len(chunks))
	for n, chunk := range chunks {
		texts[n] = chunk.Text
	}
	vectors, err := i.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed chunks: %w", err)
	}
	for n := range chunks {
		chunks[n].Vector = vectors[n]
	}
	i.chunks = append(i.chunks, chunks...)
	return nil
}

func (i *MemoryIndex) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 5
	}
	vectors, err := i.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	queryVector := vectors[0]
	hits := make([]SearchHit, 0, len(i.chunks))
	for _, chunk := range i.chunks {
		hits = append(hits, SearchHit{Chunk: chunk, Score: cosine(queryVector, chunk.Vector)})
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].Score > hits[b].Score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, aa, bb float64
	for idx := range a {
		dot += a[idx] * b[idx]
		aa += a[idx] * a[idx]
		bb += b[idx] * b[idx]
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
