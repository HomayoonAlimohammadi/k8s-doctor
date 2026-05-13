package rag

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Document struct {
	Source string
	Path   string
	Text   string
}

type Source interface {
	Load(ctx context.Context) ([]Document, error)
}

type DirectorySource struct {
	name string
	root string
}

func NewDirectorySource(name, root string) DirectorySource {
	slog.Debug("rag.NewDirectorySource", "component", "rag.source", "name", name, "root", root)
	return DirectorySource{name: name, root: root}
}

func (s DirectorySource) Load(ctx context.Context) ([]Document, error) {
	slog.Info("rag.DirectorySource.Load start", "component", "rag.source", "name", s.name, "root", s.root)
	if _, err := os.Stat(s.root); err != nil {
		slog.Warn("rag.DirectorySource.Load root not accessible (returning empty doc set)",
			"component", "rag.source", "name", s.name, "root", s.root, "error", err)
		// Returning the original error preserves existing behavior; callers
		// decide whether to treat missing docs as fatal.
	}
	var docs []Document
	skipped := 0
	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Warn("rag.DirectorySource.Load walk error",
				"component", "rag.source", "name", s.name, "path", path, "error", err)
			return err
		}
		select {
		case <-ctx.Done():
			slog.Warn("rag.DirectorySource.Load cancelled",
				"component", "rag.source", "name", s.name, "error", ctx.Err())
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".markdown") {
			skipped++
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			slog.Error("rag.DirectorySource.Load read failed",
				"component", "rag.source", "name", s.name, "path", path, "error", err)
			return fmt.Errorf("read doc %s: %w", path, err)
		}
		slog.Debug("rag.DirectorySource.Load doc",
			"component", "rag.source", "name", s.name, "path", path, "bytes", len(raw))
		docs = append(docs, Document{Source: s.name, Path: path, Text: string(raw)})
		return nil
	})
	if err != nil {
		slog.Error("rag.DirectorySource.Load failed",
			"component", "rag.source", "name", s.name, "root", s.root, "error", err)
		return nil, fmt.Errorf("load directory source %s: %w", s.root, err)
	}
	slog.Info("rag.DirectorySource.Load complete",
		"component", "rag.source", "name", s.name, "root", s.root,
		"docs", len(docs), "skipped_non_markdown", skipped)
	return docs, nil
}
