package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/audit"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/config"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/doctor"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/lab"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/llm"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/playbooks"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/rag"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools/kubectl"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools/lxd"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/web"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "k8s-doc: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		runner := tools.ExecRunner{}
		lxdClient := lxd.NewClient(runner, lxd.Config{Remote: cfg.LXDRemote, Image: cfg.LXDImage, Profiles: cfg.LXDProfiles})
		labManager := lab.NewManager(lxdBackend{client: lxdClient}, lab.Config{Name: cfg.LabName, StateDir: cfg.StateDir})
		auditLogger := audit.NewLogger(filepath.Join(cfg.StateDir, "audit.jsonl"))
		labSvc := web.RealLabService{Manager: labManager, Audit: auditLogger}

		index := rag.NewMemoryIndex(llm.FakeEmbeddingModel{})
		ragRetriever := doctor.RAGRetriever{Index: index}
		kcl := kubectl.NewClient(lxdClient)
		dnsPlaybook := playbooks.NewDNSPlaybook(kcl)
		dnsDiag := doctor.KubectlDNSDiagnostic{Playbook: dnsPlaybook}
		doctorSvc := doctor.Doctor{Retriever: ragRetriever, DNS: dnsDiag}
		doctorSvcWrapper := web.RealDoctorService{Doctor: doctorSvc, Audit: auditLogger}
		server := web.NewServer(web.Deps{Lab: labSvc, Doctor: doctorSvcWrapper})
		fmt.Printf("k8s-doc listening on http://%s\n", cfg.HTTPAddr)
		return http.ListenAndServe(cfg.HTTPAddr, server)
	case "reindex":
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		index := rag.NewMemoryIndex(llm.FakeEmbeddingModel{})
		count, err := rag.Reindexer{
			Sources: []rag.Source{
				rag.NewDirectorySource("k8s-snap", cfg.K8sSnapDocsPath),
				rag.NewDirectorySource("upstream-kubernetes", cfg.UpstreamDocsPath),
			},
			Index:         index,
			MaxChunkChars: 1200,
		}.Reindex(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("indexed %d chunks\n", count)
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

type lxdBackend struct{ client *lxd.Client }

func (b lxdBackend) Launch(ctx context.Context, name string) error { return b.client.Launch(ctx, name) }
func (b lxdBackend) Delete(ctx context.Context, name string) error { return b.client.Delete(ctx, name) }
