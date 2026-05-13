package doctor

import (
	"context"
	"testing"
)

func TestFormatAnswerIncludesRequiredSections(t *testing.T) {
	answer := FormatAnswer(Answer{
		Summary:      "DNS is broken because CoreDNS has no running pods.",
		Diagnosis:    "CoreDNS deployment was scaled to zero.",
		Evidence:     []string{"coredns replicas: 0"},
		Fix:          "Scaled CoreDNS back to one replica.",
		Verification: "DNS probe resolved kubernetes.default.",
		Citations:    []Citation{{Source: "kubernetes", Path: "dns.md", Snippet: "DNS service discovery"}},
		ToolsRun:     []string{"dns_collect", "dns_repair", "dns_verify"},
	})
	for _, section := range []string{"Summary", "Diagnosis", "Evidence", "Fix", "Verification", "Docs references", "Tools run"} {
		if !contains(answer, section) {
			t.Fatalf("answer missing section %q:\n%s", section, answer)
		}
	}
}

func TestDoctorDiagnosesDNSWithDocsAndPlaybook(t *testing.T) {
	d := Doctor{
		Retriever: FakeRetriever{Hits: []Citation{{Source: "kubernetes", Path: "dns.md", Snippet: "CoreDNS provides cluster DNS."}}},
		DNS:       FakeDNS{Report: DNSReport{Summary: "CoreDNS pods are unavailable.", Evidence: []string{"no coredns pods"}}},
	}
	answer, err := d.DiagnoseDNS(context.Background(), "session1", "cp1", "Why is DNS broken?")
	if err != nil {
		t.Fatalf("DiagnoseDNS error: %v", err)
	}
	if !contains(answer, "CoreDNS") || !contains(answer, "Docs references") {
		t.Fatalf("unexpected answer:\n%s", answer)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || contains(s[1:], sub) || s[:len(sub)] == sub))
}
