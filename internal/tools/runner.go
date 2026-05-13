package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
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
		return CommandResult{ExitCode: -1}, fmt.Errorf("command args are empty")
	}
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
		return result, fmt.Errorf("run %q: %w", args[0], err)
	}
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
