package rag

import (
	"context"
	"fmt"
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
	return DirectorySource{name: name, root: root}
}

func (s DirectorySource) Load(ctx context.Context) ([]Document, error) {
	var docs []Document
	err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".markdown") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read doc %s: %w", path, err)
		}
		docs = append(docs, Document{Source: s.name, Path: path, Text: string(raw)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load directory source %s: %w", s.root, err)
	}
	return docs, nil
}
