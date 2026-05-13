package rag

import "testing"

func TestChunkDocumentIncludesCitationFields(t *testing.T) {
	doc := Document{Source: "k8s-snap", Path: "dns.md", Text: "# DNS\nCoreDNS config\n\n## Troubleshooting\nCheck pods"}
	chunks := ChunkDocument(doc, 40)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	if chunks[0].Source != "k8s-snap" || chunks[0].Path != "dns.md" || chunks[0].Text == "" {
		t.Fatalf("unexpected chunk: %+v", chunks[0])
	}
}
