package k8ssnap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type NodeRunner interface {
	Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error)
}

type Config struct{ SnapChannel string }

type Client struct {
	runner NodeRunner
	cfg    Config
}

func NewClient(runner NodeRunner, cfg Config) *Client {
	slog.Debug("k8ssnap.NewClient", "component", "tools.k8ssnap", "channel", cfg.SnapChannel)
	return &Client{runner: runner, cfg: cfg}
}

func (c *Client) Install(ctx context.Context, node string) (tools.CommandResult, error) {
	channel := c.cfg.SnapChannel
	if channel == "" {
		channel = "latest/stable"
	}
	slog.Info("k8ssnap.Install", "component", "tools.k8ssnap", "node", node, "channel", channel)
	return c.exec(ctx, node, []string{"sudo", "snap", "install", "k8s", "--classic", "--channel", channel})
}

func (c *Client) Bootstrap(ctx context.Context, node string) (tools.CommandResult, error) {
	slog.Info("k8ssnap.Bootstrap", "component", "tools.k8ssnap", "node", node)
	return c.exec(ctx, node, []string{"sudo", "k8s", "bootstrap"})
}

func (c *Client) Status(ctx context.Context, node string) (tools.CommandResult, error) {
	slog.Info("k8ssnap.Status", "component", "tools.k8ssnap", "node", node)
	return c.exec(ctx, node, []string{"sudo", "k8s", "status"})
}

func (c *Client) exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
	result, err := c.runner.Exec(ctx, node, command)
	if err != nil {
		slog.Error("k8ssnap exec failed",
			"component", "tools.k8ssnap", "node", node, "command", command,
			"exit", result.ExitCode, "stderr", logging.Truncate(result.Stderr, 500), "error", err)
		return result, fmt.Errorf("run k8s-snap command on %s: %w", node, err)
	}
	slog.Debug("k8ssnap exec ok",
		"component", "tools.k8ssnap", "node", node, "command", command,
		"exit", result.ExitCode, "duration_ms", result.DurationMS,
		"stdout", logging.Truncate(result.Stdout, 500))
	return result, nil
}
