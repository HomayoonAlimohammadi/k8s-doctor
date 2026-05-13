package lab

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools/lxd"
)

func TestRealLXDCreateDestroy(t *testing.T) {
	if os.Getenv("K8S_DOC_RUN_LXD_TESTS") != "1" {
		t.Skip("set K8S_DOC_RUN_LXD_TESTS=1 to run real LXD tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	client := lxd.NewClient(tools.ExecRunner{}, lxd.Config{Image: "ubuntu:22.04", Profiles: []string{"default"}})
	backend := lxdBackendForTest{client: client}
	manager := NewManager(backend, Config{Name: "k8s-doc-test", StateDir: t.TempDir()})
	if _, err := manager.Create(ctx, CreateOptions{ControlPlanes: 1}); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if err := manager.Destroy(ctx); err != nil {
		t.Fatalf("Destroy error: %v", err)
	}
}

type lxdBackendForTest struct{ client *lxd.Client }

func (b lxdBackendForTest) Launch(ctx context.Context, name string) error {
	return b.client.Launch(ctx, name)
}
func (b lxdBackendForTest) Delete(ctx context.Context, name string) error {
	return b.client.Delete(ctx, name)
}
