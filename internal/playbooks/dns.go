package playbooks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/tools"
)

type Kubectl interface {
	Get(ctx context.Context, node, resource, namespace string, allNamespaces bool) (tools.CommandResult, error)
	Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error)
	Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error)
	ApplyYAML(ctx context.Context, node, yamlPath string) (tools.CommandResult, error)
	Scale(ctx context.Context, node, resource, namespace string, replicas int) (tools.CommandResult, error)
	RunDNSProbe(ctx context.Context, node string) (tools.CommandResult, error)
}

type DNSReport struct {
	Summary  string
	Evidence map[string]string
}

type DNSPlaybook struct{ kubectl Kubectl }

func NewDNSPlaybook(kubectl Kubectl) *DNSPlaybook { return &DNSPlaybook{kubectl: kubectl} }

func (p *DNSPlaybook) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
	start := time.Now()
	slog.Info("DNSPlaybook.Collect start", "component", "playbooks.dns", "control_plane", controlPlane)
	evidence := map[string]string{}
	calls := []struct {
		key string
		fn  func() (tools.CommandResult, error)
	}{
		{"pods", func() (tools.CommandResult, error) {
			return p.kubectl.Get(ctx, controlPlane, "pods", "kube-system", false)
		}},
		{"svc", func() (tools.CommandResult, error) {
			return p.kubectl.Get(ctx, controlPlane, "svc/kube-dns", "kube-system", false)
		}},
		{"endpoints", func() (tools.CommandResult, error) {
			return p.kubectl.Get(ctx, controlPlane, "endpoints/kube-dns", "kube-system", false)
		}},
		{"configmap", func() (tools.CommandResult, error) {
			return p.kubectl.Describe(ctx, controlPlane, "configmap", "coredns", "kube-system")
		}},
	}
	for _, call := range calls {
		callStart := time.Now()
		slog.Debug("DNSPlaybook.Collect step", "component", "playbooks.dns", "step", call.key)
		result, err := call.fn()
		if err != nil {
			slog.Error("DNSPlaybook.Collect step failed",
				"component", "playbooks.dns", "step", call.key,
				"duration_ms", time.Since(callStart).Milliseconds(),
				"exit", result.ExitCode, "stderr", logging.Truncate(result.Stderr, 500), "error", err)
			return DNSReport{}, fmt.Errorf("collect DNS %s: %w", call.key, err)
		}
		slog.Debug("DNSPlaybook.Collect step ok",
			"component", "playbooks.dns", "step", call.key,
			"duration_ms", time.Since(callStart).Milliseconds(),
			"exit", result.ExitCode, "stdout_chars", len(result.Stdout))
		evidence[call.key] = result.Stdout
	}
	slog.Info("DNSPlaybook.Collect complete",
		"component", "playbooks.dns", "control_plane", controlPlane,
		"keys", len(evidence), "duration_ms", time.Since(start).Milliseconds())
	return DNSReport{Summary: "Collected CoreDNS pods, service, endpoints, and ConfigMap evidence.", Evidence: evidence}, nil
}

func (p *DNSPlaybook) BreakByScalingToZero(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	slog.Warn("DNSPlaybook.BreakByScalingToZero (DESTRUCTIVE)",
		"component", "playbooks.dns", "control_plane", controlPlane)
	return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 0)
}

func (p *DNSPlaybook) RepairByScalingToOne(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	slog.Info("DNSPlaybook.RepairByScalingToOne",
		"component", "playbooks.dns", "control_plane", controlPlane)
	return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 1)
}

func (p *DNSPlaybook) Verify(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	slog.Info("DNSPlaybook.Verify (running DNS probe)",
		"component", "playbooks.dns", "control_plane", controlPlane)
	return p.kubectl.RunDNSProbe(ctx, controlPlane)
}
