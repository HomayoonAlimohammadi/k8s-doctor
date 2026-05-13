package lxd

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type Config struct {
	Remote   string
	Image    string
	Profiles []string
}

type Client struct {
	runner tools.Runner
	cfg    Config
}

var safeInstanceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,62}$`)

func NewClient(runner tools.Runner, cfg Config) *Client {
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = []string{"default"}
	}
	slog.Debug("lxd.NewClient",
		"component", "tools.lxd", "remote", cfg.Remote, "image", cfg.Image, "profiles", cfg.Profiles)
	return &Client{runner: runner, cfg: cfg}
}

func (c *Client) Launch(ctx context.Context, name string) error {
	if err := validateInstanceName(name); err != nil {
		slog.Error("lxd.Launch invalid name", "component", "tools.lxd", "name", name, "error", err)
		return err
	}
	image := c.cfg.Image
	if image == "" {
		image = "ubuntu:22.04"
	}
	args := []string{"lxc", "launch", image, name}
	for _, p := range c.cfg.Profiles {
		args = append(args, "-p", p)
	}
	slog.Info("lxd.Launch", "component", "tools.lxd", "instance", name, "image", image, "profiles", c.cfg.Profiles)
	_, err := c.runner.Run(ctx, args, tools.RunOptions{})
	if err != nil {
		slog.Error("lxd.Launch failed", "component", "tools.lxd", "instance", name, "error", err)
		return fmt.Errorf("launch LXD instance %q: %w", name, err)
	}
	slog.Info("lxd.Launch ok", "component", "tools.lxd", "instance", name)
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	if err := validateInstanceName(name); err != nil {
		slog.Error("lxd.Delete invalid name", "component", "tools.lxd", "name", name, "error", err)
		return err
	}
	slog.Info("lxd.Delete", "component", "tools.lxd", "instance", name)
	_, err := c.runner.Run(ctx, []string{"lxc", "rm", name, "--force"}, tools.RunOptions{})
	if err != nil {
		slog.Error("lxd.Delete failed", "component", "tools.lxd", "instance", name, "error", err)
		return fmt.Errorf("delete LXD instance %q: %w", name, err)
	}
	slog.Info("lxd.Delete ok", "component", "tools.lxd", "instance", name)
	return nil
}

func (c *Client) Exec(ctx context.Context, name string, command []string) (tools.CommandResult, error) {
	if err := validateInstanceName(name); err != nil {
		slog.Error("lxd.Exec invalid name", "component", "tools.lxd", "name", name, "error", err)
		return tools.CommandResult{}, err
	}
	if len(command) == 0 {
		slog.Error("lxd.Exec empty command", "component", "tools.lxd", "instance", name)
		return tools.CommandResult{}, fmt.Errorf("exec command is empty")
	}
	for _, part := range command {
		if strings.TrimSpace(part) == "" {
			slog.Error("lxd.Exec empty argument", "component", "tools.lxd", "instance", name, "command", command)
			return tools.CommandResult{}, fmt.Errorf("exec command contains empty argument")
		}
	}
	slog.Debug("lxd.Exec", "component", "tools.lxd", "instance", name, "command", command)
	args := append([]string{"lxc", "exec", name, "--"}, command...)
	result, err := c.runner.Run(ctx, args, tools.RunOptions{})
	if err != nil {
		slog.Error("lxd.Exec failed",
			"component", "tools.lxd", "instance", name, "command", command,
			"exit", result.ExitCode, "error", err)
		return result, fmt.Errorf("exec in LXD instance %q: %w", name, err)
	}
	slog.Debug("lxd.Exec ok",
		"component", "tools.lxd", "instance", name, "exit", result.ExitCode,
		"duration_ms", result.DurationMS)
	return result, nil
}

func validateInstanceName(name string) error {
	if !safeInstanceName.MatchString(name) {
		return fmt.Errorf("unsafe LXD instance name %q", name)
	}
	return nil
}
