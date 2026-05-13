package doctor

import (
	"context"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/playbooks"
)

// KubectlDNSDiagnostic wraps a playbooks.Kubectl and implements DNSDiagnostic.
type KubectlDNSDiagnostic struct{ Playbook *playbooks.DNSPlaybook }

func (d KubectlDNSDiagnostic) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
	report, err := d.Playbook.Collect(ctx, controlPlane)
	if err != nil {
		return DNSReport{Summary: "DNS evidence collection failed: " + err.Error()}, nil
	}
	return DNSReport{Summary: "Collected " + report.Summary, Evidence: summaryEvidence(report.Evidence)}, nil
}

func summaryEvidence(m map[string]string) []string {
	lines := make([]string, 0, len(m))
	for key, stdout := range m {
		if len(stdout) > 500 {
			stdout = stdout[:500] + "..."
		}
		lines = append(lines, key+": "+stdout)
	}
	return lines
}
