package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectorySourceLoadsMarkdown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dns.md"), []byte("# DNS\nCoreDNS docs"), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := NewDirectorySource("k8s-snap", dir).Load(context.Background())
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(docs) != 1 || docs[0].Source != "k8s-snap" || docs[0].Text != "# DNS\nCoreDNS docs" {
		t.Fatalf("unexpected docs: %+v", docs)
	}
}
