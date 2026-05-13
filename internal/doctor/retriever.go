package doctor

import (
	"context"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/rag"
)

// RAGRetriever wraps a rag.MemoryIndex and implements Retriever.
type RAGRetriever struct{ Index *rag.MemoryIndex }

func (r RAGRetriever) Search(ctx context.Context, query string, limit int) ([]Citation, error) {
	hits, err := r.Index.Search(ctx, query, limit)
	if err != nil {
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
	return citations, nil
}
