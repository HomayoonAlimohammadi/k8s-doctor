package lab

import (
	"context"
	"fmt"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type K8sSnap interface {
	Install(ctx context.Context, node string) (tools.CommandResult, error)
	Bootstrap(ctx context.Context, node string) (tools.CommandResult, error)
	Status(ctx context.Context, node string) (tools.CommandResult, error)
}

type ClusterService struct{ K8s K8sSnap }

func (s ClusterService) Bootstrap(ctx context.Context, state State) error {
	cp, ok := FirstControlPlane(state)
	if !ok {
		return fmt.Errorf("state has no control-plane node")
	}
	if _, err := s.K8s.Install(ctx, cp.Name); err != nil {
		return err
	}
	if _, err := s.K8s.Bootstrap(ctx, cp.Name); err != nil {
		return err
	}
	if _, err := s.K8s.Status(ctx, cp.Name); err != nil {
		return err
	}
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
