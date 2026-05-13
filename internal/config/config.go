package config

import (
	"log/slog"
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
	ChatBaseURL       string
	ChatAPIKey        string
	EmbeddingProvider string
	EmbeddingModel    string
	EmbeddingBaseURL  string
	EmbeddingAPIKey   string

	K8sSnapDocsPath  string
	UpstreamDocsPath string

	LogLevel  string
	LogFormat string
}

func Load() (Config, error) {
	cfg := Config{
		LabName:           env("K8S_DOC_LAB_NAME", "k8s-doc-lab"),
		LXDRemote:         env("K8S_DOC_LXD_REMOTE", "local"),
		LXDImage:          env("K8S_DOC_LXD_IMAGE", "ubuntu:24.04"),
		LXDProfiles:       envList("K8S_DOC_LXD_PROFILES", []string{"default", "k8s-integration"}),
		SnapChannel:       env("K8S_DOC_SNAP_CHANNEL", "latest/stable"),
		StateDir:          env("K8S_DOC_STATE_DIR", ".state"),
		HTTPAddr:          env("K8S_DOC_HTTP_ADDR", "127.0.0.1:8080"),
		ChatProvider:      env("K8S_DOC_CHAT_PROVIDER", "openai"),
		ChatModel:         env("K8S_DOC_CHAT_MODEL", "gpt-4o-mini"),
		ChatBaseURL:       env("K8S_DOC_CHAT_BASE_URL", ""),
		ChatAPIKey:        env("K8S_DOC_CHAT_API_KEY", ""),
		EmbeddingProvider: env("K8S_DOC_EMBEDDING_PROVIDER", "openai"),
		EmbeddingModel:    env("K8S_DOC_EMBEDDING_MODEL", "text-embedding-3-small"),
		EmbeddingBaseURL:  env("K8S_DOC_EMBEDDING_BASE_URL", ""),
		EmbeddingAPIKey:   env("K8S_DOC_EMBEDDING_API_KEY", ""),
		K8sSnapDocsPath:   env("K8S_DOC_K8S_SNAP_DOCS", "../../docs/canonicalk8s"),
		UpstreamDocsPath:  env("K8S_DOC_UPSTREAM_K8S_DOCS", ".state/upstream-kubernetes-docs"),
		LogLevel:          env("K8S_DOC_LOG_LEVEL", "debug"),
		LogFormat:         env("K8S_DOC_LOG_FORMAT", "auto"),
	}
	slog.Debug("config loaded",
		"component", "config",
		"lab_name", cfg.LabName,
		"lxd_remote", cfg.LXDRemote,
		"lxd_image", cfg.LXDImage,
		"lxd_profiles", cfg.LXDProfiles,
		"snap_channel", cfg.SnapChannel,
		"state_dir", cfg.StateDir,
		"http_addr", cfg.HTTPAddr,
		"chat_provider", cfg.ChatProvider,
		"chat_model", cfg.ChatModel,
		"chat_base_url", cfg.ChatBaseURL,
		"chat_api_key_set", cfg.ChatAPIKey != "",
		"embedding_provider", cfg.EmbeddingProvider,
		"embedding_model", cfg.EmbeddingModel,
		"embedding_base_url", cfg.EmbeddingBaseURL,
		"embedding_api_key_set", cfg.EmbeddingAPIKey != "",
		"k8s_snap_docs", cfg.K8sSnapDocsPath,
		"upstream_docs", cfg.UpstreamDocsPath,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
	)
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		slog.Debug("config env override", "component", "config", "key", key, "len", len(value))
		return value
	}
	slog.Debug("config env default", "component", "config", "key", key, "default", fallback)
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
			slog.Debug("config env list override", "component", "config", "key", key, "values", result)
			return result
		}
	}
	slog.Debug("config env list default", "component", "config", "key", key, "default", fallback)
	return fallback
}
