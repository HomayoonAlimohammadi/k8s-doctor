package rag

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
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
	slog.Debug("rag.NewMemoryIndex", "component", "rag.index")
	return &MemoryIndex{embedder: embedder}
}

func (i *MemoryIndex) Add(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		slog.Debug("rag.MemoryIndex.Add no-op", "component", "rag.index")
		return nil
	}
	start := time.Now()
	slog.Debug("rag.MemoryIndex.Add embed start", "component", "rag.index", "chunks", len(chunks))
	texts := make([]string, len(chunks))
	for n, chunk := range chunks {
		texts[n] = chunk.Text
	}
	vectors, err := i.embedder.Embed(ctx, texts)
	if err != nil {
		slog.Error("rag.MemoryIndex.Add embed failed",
			"component", "rag.index", "chunks", len(chunks),
			"duration_ms", time.Since(start).Milliseconds(), "error", err)
		return fmt.Errorf("embed chunks: %w", err)
	}
	for n := range chunks {
		chunks[n].Vector = vectors[n]
	}
	i.chunks = append(i.chunks, chunks...)
	slog.Debug("rag.MemoryIndex.Add ok",
		"component", "rag.index", "added", len(chunks), "total", len(i.chunks),
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}

func (i *MemoryIndex) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	start := time.Now()
	if limit <= 0 {
		limit = 5
	}
	slog.Debug("rag.MemoryIndex.Search",
		"component", "rag.index", "limit", limit, "indexed", len(i.chunks),
		"query_chars", len(query), "preview", logging.Truncate(query, 200))
	vectors, err := i.embedder.Embed(ctx, []string{query})
	if err != nil {
		slog.Error("rag.MemoryIndex.Search embed query failed",
			"component", "rag.index", "error", err)
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
	if len(hits) > 0 {
		slog.Debug("rag.MemoryIndex.Search ok",
			"component", "rag.index", "hits", len(hits),
			"top_score", hits[0].Score, "top_path", hits[0].Chunk.Path,
			"duration_ms", time.Since(start).Milliseconds())
	} else {
		slog.Warn("rag.MemoryIndex.Search returned 0 hits",
			"component", "rag.index", "indexed", len(i.chunks),
			"duration_ms", time.Since(start).Milliseconds())
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
