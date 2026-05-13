package lab

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type K8sSnap interface {
	Install(ctx context.Context, node string) (tools.CommandResult, error)
	Bootstrap(ctx context.Context, node string) (tools.CommandResult, error)
	Status(ctx context.Context, node string) (tools.CommandResult, error)
}

type ClusterService struct{ K8s K8sSnap }

func (s ClusterService) Bootstrap(ctx context.Context, state State) error {
	start := time.Now()
	slog.Info("ClusterService.Bootstrap start", "component", "lab.cluster", "lab", state.Name)
	cp, ok := FirstControlPlane(state)
	if !ok {
		slog.Error("ClusterService.Bootstrap no control-plane in state",
			"component", "lab.cluster", "lab", state.Name)
		return fmt.Errorf("state has no control-plane node")
	}
	slog.Info("ClusterService.Bootstrap installing k8s snap",
		"component", "lab.cluster", "node", cp.Name)
	res, err := s.K8s.Install(ctx, cp.Name)
	if err != nil {
		slog.Error("ClusterService.Bootstrap install failed",
			"component", "lab.cluster", "node", cp.Name,
			"exit", res.ExitCode, "stderr", logging.Truncate(res.Stderr, 500), "error", err)
		return err
	}
	slog.Info("ClusterService.Bootstrap install ok",
		"component", "lab.cluster", "node", cp.Name, "exit", res.ExitCode)

	slog.Info("ClusterService.Bootstrap bootstrapping", "component", "lab.cluster", "node", cp.Name)
	res, err = s.K8s.Bootstrap(ctx, cp.Name)
	if err != nil {
		slog.Error("ClusterService.Bootstrap bootstrap failed",
			"component", "lab.cluster", "node", cp.Name,
			"exit", res.ExitCode, "stderr", logging.Truncate(res.Stderr, 500), "error", err)
		return err
	}
	slog.Info("ClusterService.Bootstrap bootstrap ok",
		"component", "lab.cluster", "node", cp.Name, "exit", res.ExitCode)

	slog.Info("ClusterService.Bootstrap checking status", "component", "lab.cluster", "node", cp.Name)
	res, err = s.K8s.Status(ctx, cp.Name)
	if err != nil {
		slog.Error("ClusterService.Bootstrap status failed",
			"component", "lab.cluster", "node", cp.Name,
			"exit", res.ExitCode, "stderr", logging.Truncate(res.Stderr, 500), "error", err)
		return err
	}
	slog.Info("ClusterService.Bootstrap complete",
		"component", "lab.cluster", "lab", state.Name, "node", cp.Name,
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}

func FirstControlPlane(state State) (Node, bool) {
	for _, node := range state.Nodes {
		if node.Role == RoleControlPlane {
			return node, true
		}
	}
	return Node{}, false
}
