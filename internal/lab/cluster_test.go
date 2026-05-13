package lab

import (
	"context"
	"testing"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type fakeK8sSnap struct {
	install []string
	boot    []string
	status  []string
}

func (f *fakeK8sSnap) Install(ctx context.Context, node string) (tools.CommandResult, error) {
	f.install = append(f.install, node)
	return tools.CommandResult{Stdout: "installed"}, nil
}
func (f *fakeK8sSnap) Bootstrap(ctx context.Context, node string) (tools.CommandResult, error) {
	f.boot = append(f.boot, node)
	return tools.CommandResult{Stdout: "booted"}, nil
}
func (f *fakeK8sSnap) Status(ctx context.Context, node string) (tools.CommandResult, error) {
	f.status = append(f.status, node)
	return tools.CommandResult{Stdout: "ok"}, nil
}

func TestClusterBootstrapInstallsAndBootsControlPlane(t *testing.T) {
	state := State{Nodes: []Node{{Name: "demo-cp-1", Role: RoleControlPlane}}}
	f := &fakeK8sSnap{}
	svc := ClusterService{K8s: f}
	if err := svc.Bootstrap(context.Background(), state); err != nil {
		t.Fatalf("Bootstrap error: %v", err)
	}
	if len(f.install) != 1 || f.install[0] != "demo-cp-1" {
		t.Fatalf("unexpected install calls: %v", f.install)
	}
	if len(f.boot) != 1 || f.boot[0] != "demo-cp-1" {
		t.Fatalf("unexpected boot calls: %v", f.boot)
	}
	if len(f.status) != 1 || f.status[0] != "demo-cp-1" {
		t.Fatalf("unexpected status calls: %v", f.status)
	}
}
