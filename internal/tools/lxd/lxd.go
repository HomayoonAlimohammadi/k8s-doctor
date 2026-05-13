package lxd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
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
	return &Client{runner: runner, cfg: cfg}
}

func (c *Client) Launch(ctx context.Context, name string) error {
	if err := validateInstanceName(name); err != nil {
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
	_, err := c.runner.Run(ctx, args, tools.RunOptions{})
	if err != nil {
		return fmt.Errorf("launch LXD instance %q: %w", name, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
	if err := validateInstanceName(name); err != nil {
		return err
	}
	_, err := c.runner.Run(ctx, []string{"lxc", "rm", name, "--force"}, tools.RunOptions{})
	if err != nil {
		return fmt.Errorf("delete LXD instance %q: %w", name, err)
	}
	return nil
}

func (c *Client) Exec(ctx context.Context, name string, command []string) (tools.CommandResult, error) {
	if err := validateInstanceName(name); err != nil {
		return tools.CommandResult{}, err
	}
	if len(command) == 0 {
		return tools.CommandResult{}, fmt.Errorf("exec command is empty")
	}
	for _, part := range command {
		if strings.TrimSpace(part) == "" {
			return tools.CommandResult{}, fmt.Errorf("exec command contains empty argument")
		}
	}
	args := append([]string{"lxc", "exec", name, "--"}, command...)
	result, err := c.runner.Run(ctx, args, tools.RunOptions{})
	if err != nil {
		return result, fmt.Errorf("exec in LXD instance %q: %w", name, err)
	}
	return result, nil
}

func validateInstanceName(name string) error {
	if !safeInstanceName.MatchString(name) {
		return fmt.Errorf("unsafe LXD instance name %q", name)
	}
	return nil
}
