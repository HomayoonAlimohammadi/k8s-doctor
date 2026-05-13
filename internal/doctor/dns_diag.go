package doctor

import (
	"context"
	"log/slog"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/playbooks"
)

// KubectlDNSDiagnostic wraps a playbooks.Kubectl and implements DNSDiagnostic.
type KubectlDNSDiagnostic struct{ Playbook *playbooks.DNSPlaybook }

func (d KubectlDNSDiagnostic) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
	slog.Debug("KubectlDNSDiagnostic.Collect start",
		"component", "doctor.dns", "control_plane", controlPlane)
	report, err := d.Playbook.Collect(ctx, controlPlane)
	if err != nil {
		slog.Warn("KubectlDNSDiagnostic.Collect playbook error (returning soft failure)",
			"component", "doctor.dns", "control_plane", controlPlane, "error", err)
		return DNSReport{Summary: "DNS evidence collection failed: " + err.Error()}, nil
	}
	slog.Debug("KubectlDNSDiagnostic.Collect ok",
		"component", "doctor.dns", "control_plane", controlPlane, "evidence_keys", len(report.Evidence))
	return DNSReport{Summary: "Collected " + report.Summary, Evidence: summaryEvidence(report.Evidence)}, nil
}

func summaryEvidence(m map[string]string) []string {
	lines := make([]string, 0, len(m))
	for key, stdout := range m {
		lines = append(lines, key+": "+logging.Truncate(stdout, 500))
	}
	return lines
}
