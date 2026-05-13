package playbooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type fakeKubectl struct{ calls []string }

func (f *fakeKubectl) Get(ctx context.Context, node, resource, namespace string, all bool) (tools.CommandResult, error) {
	f.calls = append(f.calls, "get "+resource)
	return tools.CommandResult{Stdout: "ok"}, nil
}
func (f *fakeKubectl) Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error) {
	f.calls = append(f.calls, "describe "+resource)
	return tools.CommandResult{Stdout: "ok"}, nil
}
func (f *fakeKubectl) Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error) {
	f.calls = append(f.calls, "logs "+pod)
	return tools.CommandResult{Stdout: "ok"}, nil
}
func (f *fakeKubectl) ApplyYAML(ctx context.Context, node, yamlPath string) (tools.CommandResult, error) {
	f.calls = append(f.calls, "apply "+yamlPath)
	return tools.CommandResult{Stdout: "ok"}, nil
}
func (f *fakeKubectl) Scale(ctx context.Context, node, resource, namespace string, replicas int) (tools.CommandResult, error) {
	f.calls = append(f.calls, fmt.Sprintf("scale %s %d", resource, replicas))
	return tools.CommandResult{Stdout: "ok"}, nil
}
func (f *fakeKubectl) RunDNSProbe(ctx context.Context, node string) (tools.CommandResult, error) {
	f.calls = append(f.calls, "probe")
	return tools.CommandResult{Stdout: "ok"}, nil
}

func TestDNSCollectCallsExpectedResources(t *testing.T) {
	kube := &fakeKubectl{}
	pb := NewDNSPlaybook(kube)
	report, err := pb.Collect(context.Background(), "cp1")
	if err != nil {
		t.Fatalf("Collect error: %v", err)
	}
	if report.Summary == "" || len(kube.calls) < 4 {
		t.Fatalf("unexpected report/calls: %+v %#v", report, kube.calls)
	}
}
