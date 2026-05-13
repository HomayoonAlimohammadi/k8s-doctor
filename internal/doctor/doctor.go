package doctor

import (
	"context"
	"fmt"
	"strings"
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
	citations, err := d.Retriever.Search(ctx, question, 5)
	if err != nil {
		return "", fmt.Errorf("search docs: %w", err)
	}
	report, err := d.DNS.Collect(ctx, controlPlane)
	if err != nil {
		return "", fmt.Errorf("collect DNS diagnostics: %w", err)
	}
	return FormatAnswer(Answer{
		Summary:      "DNS appears unhealthy based on live CoreDNS evidence.",
		Diagnosis:    report.Summary,
		Evidence:     report.Evidence,
		Fix:          "For the MVP DNS scenario, restore CoreDNS to a healthy replica count and re-run the DNS probe.",
		Verification: "Run dns_verify after repair to confirm kubernetes.default resolves from a test pod.",
		Citations:    citations,
		ToolsRun:     []string{"docs_search", "dns_collect"},
	}), nil
}

type FakeRetriever struct{ Hits []Citation }

func (f FakeRetriever) Search(ctx context.Context, query string, limit int) ([]Citation, error) {
	return f.Hits, nil
}

type FakeDNS struct{ Report DNSReport }

func (f FakeDNS) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
	return f.Report, nil
}
