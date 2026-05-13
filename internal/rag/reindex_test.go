package rag

import (
	"context"
	"testing"
)

func TestReindexLoadsSourcesAndIndexesChunks(t *testing.T) {
	idx := NewMemoryIndex(fakeEmbedder{})
	reindexer := Reindexer{
		Sources: []Source{fakeSource{docs: []Document{{Source: "test", Path: "dns.md", Text: "# DNS\nCoreDNS docs"}}}},
		Index:   idx,
	}
	count, err := reindexer.Reindex(context.Background())
	if err != nil {
		t.Fatalf("Reindex error: %v", err)
	}
	if count == 0 {
		t.Fatal("expected indexed chunks")
	}
}

type fakeSource struct{ docs []Document }

func (f fakeSource) Load(ctx context.Context) ([]Document, error) { return f.docs, nil }
