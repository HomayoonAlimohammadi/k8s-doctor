package lxd

import (
	"context"
	"reflect"
	"testing"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

func TestLaunchUsesProfilesAndImage(t *testing.T) {
	runner := tools.NewFakeRunner(tools.CommandResult{Stdout: "", ExitCode: 0})
	client := NewClient(runner, Config{Remote: "local", Image: "ubuntu:22.04", Profiles: []string{"default", "k8s-profile"}})
	if err := client.Launch(context.Background(), "k8s-doc-lab-cp1"); err != nil {
		t.Fatalf("Launch returned error: %v", err)
	}
	got := runner.Commands()[0]
	want := []string{"lxc", "launch", "ubuntu:22.04", "k8s-doc-lab-cp1", "-p", "default", "-p", "k8s-profile"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command = %#v, want %#v", got, want)
	}
}

func TestExecRejectsEmptyCommand(t *testing.T) {
	runner := tools.NewFakeRunner(tools.CommandResult{})
	client := NewClient(runner, Config{Image: "ubuntu:22.04", Profiles: []string{"default"}})
	if _, err := client.Exec(context.Background(), "node1", nil); err == nil {
		t.Fatal("Exec returned nil error for empty command")
	}
}
