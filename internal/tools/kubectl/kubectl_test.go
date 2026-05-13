package kubectl

import (
	"context"
	"reflect"
	"testing"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type fakeNodeRunner struct{ commands [][]string }

func (f *fakeNodeRunner) Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
	f.commands = append(f.commands, append([]string{node}, command...))
	return tools.CommandResult{Stdout: "ok", ExitCode: 0}, nil
}

func TestGetPodsAllNamespaces(t *testing.T) {
	runner := &fakeNodeRunner{}
	client := NewClient(runner)
	if _, err := client.Get(context.Background(), "cp1", "pods", "", true); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	want := []string{"cp1", "sudo", "k8s", "kubectl", "get", "pods", "-A", "-o", "wide"}
	if !reflect.DeepEqual(runner.commands[0], want) {
		t.Fatalf("command = %#v, want %#v", runner.commands[0], want)
	}
}
