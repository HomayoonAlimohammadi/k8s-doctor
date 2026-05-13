package tools

import (
	"context"
	"testing"
)

func TestFakeRunnerRecordsCommands(t *testing.T) {
	runner := NewFakeRunner(CommandResult{Stdout: "ok\n", ExitCode: 0})
	result, err := runner.Run(context.Background(), []string{"echo", "ok"}, RunOptions{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Stdout != "ok\n" || result.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(runner.Commands()) != 1 || runner.Commands()[0][0] != "echo" {
		t.Fatalf("commands not recorded: %+v", runner.Commands())
	}
}
