package kubectl

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type NodeRunner interface {
	Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error)
}

type Client struct {
	runner NodeRunner
}

var safeName = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

func NewClient(runner NodeRunner) *Client {
	slog.Debug("kubectl.NewClient", "component", "tools.kubectl")
	return &Client{runner: runner}
}

func (c *Client) Get(ctx context.Context, node, resource, namespace string, allNamespaces bool) (tools.CommandResult, error) {
	if !safeName.MatchString(resource) {
		slog.Error("kubectl.Get unsafe resource", "component", "tools.kubectl", "resource", resource)
		return tools.CommandResult{}, fmt.Errorf("unsafe resource %q", resource)
	}
	args := []string{"sudo", "k8s", "kubectl", "get", resource}
	if allNamespaces {
		args = append(args, "-A")
	}
	if namespace != "" {
		if !safeName.MatchString(namespace) {
			slog.Error("kubectl.Get unsafe namespace", "component", "tools.kubectl", "namespace", namespace)
			return tools.CommandResult{}, fmt.Errorf("unsafe namespace %q", namespace)
		}
		args = append(args, "-n", namespace)
	}
	args = append(args, "-o", "wide")
	slog.Debug("kubectl.Get",
		"component", "tools.kubectl", "node", node,
		"resource", resource, "namespace", namespace, "all_namespaces", allNamespaces)
	return c.runner.Exec(ctx, node, args)
}

func (c *Client) Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error) {
	for label, value := range map[string]string{"resource": resource, "name": name, "namespace": namespace} {
		if value != "" && !safeName.MatchString(value) {
			slog.Error("kubectl.Describe unsafe input",
				"component", "tools.kubectl", "label", label, "value", value)
			return tools.CommandResult{}, fmt.Errorf("unsafe %s %q", label, value)
		}
	}
	args := []string{"sudo", "k8s", "kubectl", "describe", resource, name}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	slog.Debug("kubectl.Describe",
		"component", "tools.kubectl", "node", node,
		"resource", resource, "name", name, "namespace", namespace)
	return c.runner.Exec(ctx, node, args)
}

func (c *Client) Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error) {
	if !safeName.MatchString(pod) || !safeName.MatchString(namespace) {
		slog.Error("kubectl.Logs unsafe pod or namespace",
			"component", "tools.kubectl", "pod", pod, "namespace", namespace)
		return tools.CommandResult{}, fmt.Errorf("unsafe pod or namespace")
	}
	if tail <= 0 {
		tail = 100
	}
	slog.Debug("kubectl.Logs",
		"component", "tools.kubectl", "node", node, "pod", pod, "namespace", namespace, "tail", tail)
	return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "logs", pod, "-n", namespace, "--tail", fmt.Sprint(tail)})
}

func (c *Client) ApplyYAML(ctx context.Context, node string, yamlPath string) (tools.CommandResult, error) {
	if !safeName.MatchString(yamlPath) {
		slog.Error("kubectl.ApplyYAML unsafe yaml path",
			"component", "tools.kubectl", "path", yamlPath)
		return tools.CommandResult{}, fmt.Errorf("unsafe yaml path %q", yamlPath)
	}
	slog.Info("kubectl.ApplyYAML",
		"component", "tools.kubectl", "node", node, "path", yamlPath)
	return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "apply", "-f", yamlPath})
}

func (c *Client) Scale(ctx context.Context, node, resource, namespace string, replicas int) (tools.CommandResult, error) {
	if !safeName.MatchString(resource) || !safeName.MatchString(namespace) {
		slog.Error("kubectl.Scale unsafe scale target",
			"component", "tools.kubectl", "resource", resource, "namespace", namespace)
		return tools.CommandResult{}, fmt.Errorf("unsafe scale target")
	}
	slog.Info("kubectl.Scale",
		"component", "tools.kubectl", "node", node, "resource", resource,
		"namespace", namespace, "replicas", replicas)
	return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "scale", resource, "-n", namespace, "--replicas", fmt.Sprint(replicas)})
}

func (c *Client) RunDNSProbe(ctx context.Context, node string) (tools.CommandResult, error) {
	slog.Info("kubectl.RunDNSProbe", "component", "tools.kubectl", "node", node)
	return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "run", "k8s-doc-dns-probe", "--image=busybox:1.36", "--restart=Never", "--rm", "-i", "--", "nslookup", "kubernetes.default"})
}
