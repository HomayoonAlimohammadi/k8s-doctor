package rag

import (
	"context"
	"fmt"
)

type Reindexer struct {
	Sources       []Source
	Index         *MemoryIndex
	MaxChunkChars int
}

func (r Reindexer) Reindex(ctx context.Context) (int, error) {
	total := 0
	for _, source := range r.Sources {
		docs, err := source.Load(ctx)
		if err != nil {
			return total, fmt.Errorf("load source: %w", err)
		}
		for _, doc := range docs {
			chunks := ChunkDocument(doc, r.MaxChunkChars)
			if err := r.Index.Add(ctx, chunks); err != nil {
				return total, err
			}
			total += len(chunks)
		}
	}
	return total, nil
}
