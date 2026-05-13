package tools

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type RunOptions struct {
	Input   []byte
	WorkDir string
}

type CommandResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMS int64
}

type Runner interface {
	Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error) {
	if len(args) == 0 {
		slog.Error("ExecRunner.Run empty args", "component", "tools.runner")
		return CommandResult{ExitCode: -1}, fmt.Errorf("command args are empty")
	}
	slog.Debug("ExecRunner.Run start",
		"component", "tools.runner",
		"args", args, "workdir", opts.WorkDir, "input_bytes", len(opts.Input))

	start := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = opts.WorkDir
	cmd.Stdin = bytes.NewReader(opts.Input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode, DurationMS: time.Since(start).Milliseconds()}
	if err != nil {
		slog.Error("ExecRunner.Run failed",
			"component", "tools.runner",
			"args", args, "exit", exitCode, "duration_ms", result.DurationMS,
			"stdout", logging.Truncate(result.Stdout, 500),
			"stderr", logging.Truncate(result.Stderr, 500),
			"error", err)
		return result, fmt.Errorf("run %q: %w", args[0], err)
	}
	slog.Debug("ExecRunner.Run ok",
		"component", "tools.runner",
		"args", args, "exit", exitCode, "duration_ms", result.DurationMS,
		"stdout_chars", len(result.Stdout), "stderr_chars", len(result.Stderr),
		"stdout_preview", logging.Truncate(result.Stdout, 500),
		"stderr_preview", logging.Truncate(result.Stderr, 500))
	return result, nil
}

type FakeRunner struct {
	result   CommandResult
	err      error
	commands [][]string
	mu       sync.Mutex
}

func NewFakeRunner(result CommandResult) *FakeRunner {
	return &FakeRunner{result: result}
}

func NewFailingFakeRunner(result CommandResult, err error) *FakeRunner {
	return &FakeRunner{result: result, err: err}
}

func (f *FakeRunner) Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := append([]string(nil), args...)
	f.commands = append(f.commands, copied)
	slog.Debug("FakeRunner.Run",
		"component", "tools.runner.fake",
		"args", args, "would_exit", f.result.ExitCode, "err", errString(f.err))
	return f.result, f.err
}

func (f *FakeRunner) Commands() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := make([][]string, len(f.commands))
	for i := range f.commands {
		copied[i] = append([]string(nil), f.commands[i]...)
	}
	return copied
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
