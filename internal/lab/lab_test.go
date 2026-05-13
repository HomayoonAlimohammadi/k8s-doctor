package lab

import (
	"context"
	"testing"
)

type fakeBackend struct {
	created []string
	deleted []string
}

func (f *fakeBackend) Launch(ctx context.Context, name string) error {
	f.created = append(f.created, name)
	return nil
}
func (f *fakeBackend) Delete(ctx context.Context, name string) error {
	f.deleted = append(f.deleted, name)
	return nil
}

func TestCreateDefaultLab(t *testing.T) {
	backend := &fakeBackend{}
	manager := NewManager(backend, Config{Name: "demo", StateDir: t.TempDir()})
	state, err := manager.Create(context.Background(), CreateOptions{ControlPlanes: 1})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(state.Nodes) != 1 || state.Nodes[0].Name != "demo-cp-1" || state.Nodes[0].Role != RoleControlPlane {
		t.Fatalf("unexpected state: %+v", state)
	}
	if len(backend.created) != 1 || backend.created[0] != "demo-cp-1" {
		t.Fatalf("unexpected created instances: %+v", backend.created)
	}
}
