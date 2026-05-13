package k8ssnap

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

func TestBootstrapCommand(t *testing.T) {
	runner := &fakeNodeRunner{}
	client := NewClient(runner, Config{SnapChannel: "latest/stable"})
	if _, err := client.Bootstrap(context.Background(), "cp1"); err != nil {
		t.Fatalf("Bootstrap error: %v", err)
	}
	want := []string{"cp1", "sudo", "k8s", "bootstrap"}
	if !reflect.DeepEqual(runner.commands[0], want) {
		t.Fatalf("command = %#v, want %#v", runner.commands[0], want)
	}
}
