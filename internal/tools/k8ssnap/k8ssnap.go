package k8ssnap

import (
	"context"
	"fmt"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
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
	return &Client{runner: runner, cfg: cfg}
}

func (c *Client) Install(ctx context.Context, node string) (tools.CommandResult, error) {
	channel := c.cfg.SnapChannel
	if channel == "" {
		channel = "latest/stable"
	}
	return c.exec(ctx, node, []string{"sudo", "snap", "install", "k8s", "--classic", "--channel", channel})
}

func (c *Client) Bootstrap(ctx context.Context, node string) (tools.CommandResult, error) {
	return c.exec(ctx, node, []string{"sudo", "k8s", "bootstrap"})
}

func (c *Client) Status(ctx context.Context, node string) (tools.CommandResult, error) {
	return c.exec(ctx, node, []string{"sudo", "k8s", "status"})
}

func (c *Client) exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
	result, err := c.runner.Exec(ctx, node, command)
	if err != nil {
		return result, fmt.Errorf("run k8s-snap command on %s: %w", node, err)
	}
	return result, nil
}
