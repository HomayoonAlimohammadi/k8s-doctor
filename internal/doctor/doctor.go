package doctor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type Citation struct {
	Source  string
	Path    string
	Snippet string
}

type Answer struct {
	Summary      string
	Diagnosis    string
	Evidence     []string
	Fix          string
	Verification string
	Citations    []Citation
	ToolsRun     []string
}

func FormatAnswer(answer Answer) string {
	slog.Debug("FormatAnswer",
		"component", "doctor",
		"evidence", len(answer.Evidence),
		"citations", len(answer.Citations),
		"tools", len(answer.ToolsRun))
	var b strings.Builder
	b.WriteString("## Summary\n\n" + answer.Summary + "\n\n")
	b.WriteString("## Diagnosis\n\n" + answer.Diagnosis + "\n\n")
	b.WriteString("## Evidence\n\n")
	for _, item := range answer.Evidence {
		b.WriteString("- " + item + "\n")
	}
	b.WriteString("\n## Fix\n\n" + answer.Fix + "\n\n")
	b.WriteString("## Verification\n\n" + answer.Verification + "\n\n")
	b.WriteString("## Docs references\n\n")
	for _, citation := range answer.Citations {
		b.WriteString("- " + citation.Source + ": `" + citation.Path + "` — " + citation.Snippet + "\n")
	}
	b.WriteString("\n## Tools run\n\n")
	for _, tool := range answer.ToolsRun {
		b.WriteString("- " + tool + "\n")
	}
	return b.String()
}

type Retriever interface {
	Search(ctx context.Context, query string, limit int) ([]Citation, error)
}

type DNSDiagnostic interface {
	Collect(ctx context.Context, controlPlane string) (DNSReport, error)
}

type DNSReport struct {
	Summary  string
	Evidence []string
}

type Doctor struct {
	Retriever Retriever
	DNS       DNSDiagnostic
}

func (d Doctor) DiagnoseDNS(ctx context.Context, sessionID, controlPlane, question string) (string, error) {
	start := time.Now()
	slog.Info("Doctor.DiagnoseDNS start",
		"component", "doctor", "session_id", sessionID, "control_plane", controlPlane,
		"question_chars", len(question), "preview", logging.Truncate(question, 200))

	searchStart := time.Now()
	citations, err := d.Retriever.Search(ctx, question, 5)
	if err != nil {
		slog.Error("Doctor.DiagnoseDNS retriever.Search failed",
			"component", "doctor", "session_id", sessionID,
			"duration_ms", time.Since(searchStart).Milliseconds(), "error", err)
		return "", fmt.Errorf("search docs: %w", err)
	}
	slog.Info("Doctor.DiagnoseDNS retriever.Search ok",
		"component", "doctor", "session_id", sessionID, "hits", len(citations),
		"duration_ms", time.Since(searchStart).Milliseconds())

	collectStart := time.Now()
	report, err := d.DNS.Collect(ctx, controlPlane)
	if err != nil {
		slog.Error("Doctor.DiagnoseDNS dns.Collect failed",
			"component", "doctor", "session_id", sessionID,
			"duration_ms", time.Since(collectStart).Milliseconds(), "error", err)
		return "", fmt.Errorf("collect DNS diagnostics: %w", err)
	}
	slog.Info("Doctor.DiagnoseDNS dns.Collect ok",
		"component", "doctor", "session_id", sessionID,
		"evidence", len(report.Evidence),
		"duration_ms", time.Since(collectStart).Milliseconds())

	out := FormatAnswer(Answer{
		Summary:      "DNS appears unhealthy based on live CoreDNS evidence.",
		Diagnosis:    report.Summary,
		Evidence:     report.Evidence,
		Fix:          "For the MVP DNS scenario, restore CoreDNS to a healthy replica count and re-run the DNS probe.",
		Verification: "Run dns_verify after repair to confirm kubernetes.default resolves from a test pod.",
		Citations:    citations,
		ToolsRun:     []string{"docs_search", "dns_collect"},
	})
	slog.Info("Doctor.DiagnoseDNS complete",
		"component", "doctor", "session_id", sessionID,
		"answer_chars", len(out), "duration_ms", time.Since(start).Milliseconds())
	return out, nil
}

type FakeRetriever struct{ Hits []Citation }

func (f FakeRetriever) Search(ctx context.Context, query string, limit int) ([]Citation, error) {
	slog.Debug("FakeRetriever.Search", "component", "doctor.fake", "query", query, "limit", limit, "hits", len(f.Hits))
	return f.Hits, nil
}

type FakeDNS struct{ Report DNSReport }

func (f FakeDNS) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
	slog.Debug("FakeDNS.Collect", "component", "doctor.fake", "control_plane", controlPlane)
	return f.Report, nil
}
