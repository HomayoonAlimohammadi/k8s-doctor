package doctor

import (
	"context"
	"log/slog"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/rag"
)

// RAGRetriever wraps a rag.MemoryIndex and implements Retriever.
type RAGRetriever struct{ Index *rag.MemoryIndex }

func (r RAGRetriever) Search(ctx context.Context, query string, limit int) ([]Citation, error) {
	start := time.Now()
	slog.Debug("RAGRetriever.Search start",
		"component", "doctor.retriever", "limit", limit,
		"query_chars", len(query), "preview", logging.Truncate(query, 200))
	hits, err := r.Index.Search(ctx, query, limit)
	if err != nil {
		slog.Error("RAGRetriever.Search failed",
			"component", "doctor.retriever", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return nil, err
	}
	citations := make([]Citation, 0, len(hits))
	for _, hit := range hits {
		citations = append(citations, Citation{
			Source:  hit.Chunk.Source,
			Path:    hit.Chunk.Path,
			Snippet: hit.Chunk.Text,
		})
	}
	slog.Debug("RAGRetriever.Search ok",
		"component", "doctor.retriever", "hits", len(citations),
		"duration_ms", time.Since(start).Milliseconds())
	return citations, nil
}
