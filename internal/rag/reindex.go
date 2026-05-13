package rag

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Reindexer struct {
	Sources       []Source
	Index         *MemoryIndex
	MaxChunkChars int
}

func (r Reindexer) Reindex(ctx context.Context) (int, error) {
	start := time.Now()
	slog.Info("rag.Reindexer.Reindex start",
		"component", "rag.reindex", "sources", len(r.Sources), "max_chunk_chars", r.MaxChunkChars)
	total := 0
	for _, source := range r.Sources {
		srcStart := time.Now()
		docs, err := source.Load(ctx)
		if err != nil {
			slog.Error("rag.Reindexer source load failed",
				"component", "rag.reindex",
				"duration_ms", time.Since(srcStart).Milliseconds(), "error", err)
			return total, fmt.Errorf("load source: %w", err)
		}
		slog.Info("rag.Reindexer source loaded",
			"component", "rag.reindex", "docs", len(docs),
			"duration_ms", time.Since(srcStart).Milliseconds())
		for _, doc := range docs {
			chunks := ChunkDocument(doc, r.MaxChunkChars)
			slog.Debug("rag.Reindexer chunked",
				"component", "rag.reindex", "source", doc.Source, "path", doc.Path,
				"chunks", len(chunks))
			if err := r.Index.Add(ctx, chunks); err != nil {
				slog.Error("rag.Reindexer add chunks failed",
					"component", "rag.reindex", "source", doc.Source, "path", doc.Path, "error", err)
				return total, err
			}
			total += len(chunks)
		}
	}
	slog.Info("rag.Reindexer.Reindex complete",
		"component", "rag.reindex", "total_chunks", total,
		"duration_ms", time.Since(start).Milliseconds())
	return total, nil
}
