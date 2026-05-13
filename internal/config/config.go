package config

import (
	"os"
	"strings"
)

type Config struct {
	LabName     string
	LXDRemote   string
	LXDImage    string
	LXDProfiles []string
	SnapChannel string
	StateDir    string
	HTTPAddr    string

	ChatProvider      string
	ChatModel         string
	EmbeddingProvider string
	EmbeddingModel    string

	K8sSnapDocsPath  string
	UpstreamDocsPath string
}

func Load() (Config, error) {
	return Config{
		LabName:           env("K8S_DOC_LAB_NAME", "k8s-doc-lab"),
		LXDRemote:         env("K8S_DOC_LXD_REMOTE", "local"),
		LXDImage:          env("K8S_DOC_LXD_IMAGE", "ubuntu:24.04"),
		LXDProfiles:       envList("K8S_DOC_LXD_PROFILES", []string{"default", "k8s-integration"}),
		SnapChannel:       env("K8S_DOC_SNAP_CHANNEL", "latest/stable"),
		StateDir:          env("K8S_DOC_STATE_DIR", ".state"),
		HTTPAddr:          env("K8S_DOC_HTTP_ADDR", "127.0.0.1:8080"),
		ChatProvider:      env("K8S_DOC_CHAT_PROVIDER", "openai"),
		ChatModel:         env("K8S_DOC_CHAT_MODEL", "gpt-4o-mini"),
		EmbeddingProvider: env("K8S_DOC_EMBEDDING_PROVIDER", "openai"),
		EmbeddingModel:    env("K8S_DOC_EMBEDDING_MODEL", "text-embedding-3-small"),
		K8sSnapDocsPath:   env("K8S_DOC_K8S_SNAP_DOCS", "../../docs/canonicalk8s"),
		UpstreamDocsPath:  env("K8S_DOC_UPSTREAM_K8S_DOCS", ".state/upstream-kubernetes-docs"),
	}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envList(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}
