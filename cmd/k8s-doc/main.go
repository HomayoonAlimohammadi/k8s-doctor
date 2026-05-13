package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/audit"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/config"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/doctor"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/lab"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/llm"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/playbooks"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/rag"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools/kubectl"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools/lxd"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/web"
)

func main() {
	// Bootstrap logger from env so that even config.Load logs are visible at
	// the level the user asked for. config.Load will log the resolved values
	// again for the record.
	logging.Setup(os.Getenv("K8S_DOC_LOG_LEVEL"), os.Getenv("K8S_DOC_LOG_FORMAT"), os.Stderr)
	slog.Info("k8s-doc starting", "component", "main", "pid", os.Getpid(), "args", os.Args)

	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("k8s-doc fatal", "component", "main", "error", err)
		fmt.Fprintf(os.Stderr, "k8s-doc: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}
	slog.Info("dispatching command", "component", "main", "command", command, "args", args)

	switch command {
	case "serve":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		// Re-apply log config in case the env vars were not yet present at
		// the bootstrap call above (config.Load resolves defaults too).
		logging.Setup(cfg.LogLevel, cfg.LogFormat, os.Stderr)

		slog.Info("wiring tools.ExecRunner", "component", "main")
		runner := tools.ExecRunner{}

		slog.Info("wiring lxd client", "component", "main",
			"remote", cfg.LXDRemote, "image", cfg.LXDImage, "profiles", cfg.LXDProfiles)
		lxdClient := lxd.NewClient(runner, lxd.Config{Remote: cfg.LXDRemote, Image: cfg.LXDImage, Profiles: cfg.LXDProfiles})

		slog.Info("wiring lab manager", "component", "main",
			"name", cfg.LabName, "state_dir", cfg.StateDir)
		labManager := lab.NewManager(lxdBackend{client: lxdClient}, lab.Config{Name: cfg.LabName, StateDir: cfg.StateDir})

		auditPath := filepath.Join(cfg.StateDir, "audit.jsonl")
		slog.Info("wiring audit logger", "component", "main", "path", auditPath)
		auditLogger := audit.NewLogger(auditPath)

		labSvc := web.RealLabService{Manager: labManager, Audit: auditLogger}

		slog.Info("wiring rag memory index", "component", "main",
			"embedding_provider", cfg.EmbeddingProvider, "embedding_model", cfg.EmbeddingModel)
		embedder := selectEmbedder(cfg)
		index := rag.NewMemoryIndex(embedder)

		// Surface chat provider too, even though the doctor MVP does not yet
		// drive a chat model. The user wants this clearly visible.
		_ = selectChat(cfg)

		ragRetriever := doctor.RAGRetriever{Index: index}
		slog.Info("wiring kubectl client", "component", "main")
		kcl := kubectl.NewClient(lxdClient)

		slog.Info("wiring dns playbook", "component", "main")
		dnsPlaybook := playbooks.NewDNSPlaybook(kcl)
		dnsDiag := doctor.KubectlDNSDiagnostic{Playbook: dnsPlaybook}

		slog.Info("wiring doctor service", "component", "main")
		doctorSvc := doctor.Doctor{Retriever: ragRetriever, DNS: dnsDiag}
		doctorSvcWrapper := web.RealDoctorService{Doctor: doctorSvc, Audit: auditLogger}

		slog.Info("wiring web server", "component", "main", "addr", cfg.HTTPAddr)
		server := web.NewServer(web.Deps{Lab: labSvc, Doctor: doctorSvcWrapper})

		slog.Info("k8s-doc listening", "component", "main", "addr", cfg.HTTPAddr, "url", "http://"+cfg.HTTPAddr)
		fmt.Printf("k8s-doc listening on http://%s\n", cfg.HTTPAddr)
		if err := http.ListenAndServe(cfg.HTTPAddr, server); err != nil {
			slog.Error("http server stopped", "component", "main", "error", err)
			return err
		}
		return nil
	case "reindex":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logging.Setup(cfg.LogLevel, cfg.LogFormat, os.Stderr)
		slog.Info("reindex start", "component", "main",
			"k8s_snap_docs", cfg.K8sSnapDocsPath, "upstream_docs", cfg.UpstreamDocsPath,
			"embedding_provider", cfg.EmbeddingProvider, "embedding_model", cfg.EmbeddingModel)
		embedder := selectEmbedder(cfg)
		index := rag.NewMemoryIndex(embedder)
		count, err := rag.Reindexer{
			Sources: []rag.Source{
				rag.NewDirectorySource("k8s-snap", cfg.K8sSnapDocsPath),
				rag.NewDirectorySource("upstream-kubernetes", cfg.UpstreamDocsPath),
			},
			Index:         index,
			MaxChunkChars: 1200,
		}.Reindex(ctx)
		if err != nil {
			slog.Error("reindex failed", "component", "main", "error", err)
			return err
		}
		slog.Info("reindex complete", "component", "main", "chunks", count)
		fmt.Printf("indexed %d chunks\n", count)
		return nil
	default:
		slog.Error("unknown command", "component", "main", "command", command)
		return fmt.Errorf("unknown command %q", command)
	}
}

// selectEmbedder picks the embedding implementation based on config. Falls
// back to FakeEmbeddingModel with a loud warning, so the user can see in the
// logs exactly when real embeddings are not in use.
func selectEmbedder(cfg config.Config) rag.Embedder {
	provider := strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider))
	switch provider {
	case "openai":
		if cfg.EmbeddingAPIKey == "" {
			slog.Warn("embedding provider 'openai' requested but K8S_DOC_EMBEDDING_API_KEY is empty; falling back to FakeEmbeddingModel",
				"component", "main", "embedding_provider", cfg.EmbeddingProvider, "embedding_model", cfg.EmbeddingModel)
			return llm.FakeEmbeddingModel{}
		}
		slog.Info("embedding provider: openai", "component", "main",
			"model", cfg.EmbeddingModel, "base_url", cfg.EmbeddingBaseURL,
			"api_key", logging.RedactBearer(cfg.EmbeddingAPIKey))
		return llm.OpenAIEmbedder{BaseURL: cfg.EmbeddingBaseURL, APIKey: cfg.EmbeddingAPIKey, Model: cfg.EmbeddingModel}
	case "ollama":
		slog.Info("embedding provider: ollama", "component", "main",
			"model", cfg.EmbeddingModel, "base_url", cfg.EmbeddingBaseURL)
		return llm.OllamaEmbedder{BaseURL: cfg.EmbeddingBaseURL, Model: cfg.EmbeddingModel}
	case "fake", "":
		slog.Warn("embedding provider: fake (no real embeddings)", "component", "main")
		return llm.FakeEmbeddingModel{}
	default:
		slog.Warn("unknown embedding provider; falling back to FakeEmbeddingModel",
			"component", "main", "provider", cfg.EmbeddingProvider)
		return llm.FakeEmbeddingModel{}
	}
}

// selectChat is currently informational: it logs what the chat provider
// would be, so the user can see that the MVP doctor flow does not yet route
// through it. Returning the model anyway keeps wiring close to ready.
func selectChat(cfg config.Config) llm.ChatModel {
	provider := strings.ToLower(strings.TrimSpace(cfg.ChatProvider))
	slog.Warn("chat model is configured but the MVP doctor flow does not route through it yet",
		"component", "main", "chat_provider", cfg.ChatProvider, "chat_model", cfg.ChatModel)
	switch provider {
	case "openai":
		if cfg.ChatAPIKey == "" {
			slog.Warn("chat provider 'openai' requested but K8S_DOC_CHAT_API_KEY is empty; would use FakeChatModel",
				"component", "main")
			return llm.FakeChatModel{Response: "fake chat response"}
		}
		return llm.OpenAIClient{BaseURL: cfg.ChatBaseURL, APIKey: cfg.ChatAPIKey, Model: cfg.ChatModel}
	case "ollama":
		return llm.OllamaClient{BaseURL: cfg.ChatBaseURL, Model: cfg.ChatModel}
	case "fake", "":
		return llm.FakeChatModel{Response: "fake chat response"}
	default:
		slog.Warn("unknown chat provider", "component", "main", "provider", cfg.ChatProvider)
		return llm.FakeChatModel{Response: "fake chat response"}
	}
}

type lxdBackend struct{ client *lxd.Client }

func (b lxdBackend) Launch(ctx context.Context, name string) error { return b.client.Launch(ctx, name) }
func (b lxdBackend) Delete(ctx context.Context, name string) error { return b.client.Delete(ctx, name) }
