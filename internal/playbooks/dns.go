package playbooks

import (
	"context"
	"fmt"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
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
		result, err := call.fn()
		if err != nil {
			return DNSReport{}, fmt.Errorf("collect DNS %s: %w", call.key, err)
		}
		evidence[call.key] = result.Stdout
	}
	return DNSReport{Summary: "Collected CoreDNS pods, service, endpoints, and ConfigMap evidence.", Evidence: evidence}, nil
}

func (p *DNSPlaybook) BreakByScalingToZero(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 0)
}

func (p *DNSPlaybook) RepairByScalingToOne(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 1)
}

func (p *DNSPlaybook) Verify(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
	return p.kubectl.RunDNSProbe(ctx, controlPlane)
}
