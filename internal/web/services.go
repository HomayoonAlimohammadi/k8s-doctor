package web

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/audit"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/lab"
	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type AuditLogger interface {
	Record(ctx context.Context, entry audit.Entry) error
}

type LabManager interface {
	Create(ctx context.Context, opts lab.CreateOptions) (lab.State, error)
	Destroy(ctx context.Context) error
	Load() (lab.State, error)
}

type RealLabService struct {
	Manager LabManager
	Audit   AuditLogger
}

func (s RealLabService) CreateOrReuse(r *http.Request) (string, error) {
	start := time.Now()
	slog.Info("RealLabService.CreateOrReuse start", "component", "web.services")
	state, err := s.Manager.Create(r.Context(), lab.CreateOptions{ControlPlanes: 1})
	if err != nil {
		slog.Error("RealLabService.CreateOrReuse failed",
			"component", "web.services", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", err
	}
	msg := "lab ready: " + state.Name
	slog.Info("RealLabService.CreateOrReuse ok",
		"component", "web.services", "lab", state.Name, "nodes", len(state.Nodes),
		"duration_ms", time.Since(start).Milliseconds())
	if s.Audit != nil {
		if err := s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "lab_create", Result: msg}); err != nil {
			slog.Warn("audit record failed", "component", "web.services", "tool", "lab_create", "error", err)
		}
	}
	return msg, nil
}

func (s RealLabService) Destroy(r *http.Request) (string, error) {
	start := time.Now()
	slog.Info("RealLabService.Destroy start", "component", "web.services")
	if err := s.Manager.Destroy(r.Context()); err != nil {
		slog.Error("RealLabService.Destroy failed",
			"component", "web.services", "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return "", err
	}
	msg := "lab destroyed"
	slog.Info("RealLabService.Destroy ok",
		"component", "web.services", "duration_ms", time.Since(start).Milliseconds())
	if s.Audit != nil {
		if err := s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "lab_destroy", Result: msg}); err != nil {
			slog.Warn("audit record failed", "component", "web.services", "tool", "lab_destroy", "error", err)
		}
	}
	return msg, nil
}

func (s RealLabService) Status(r *http.Request) (string, error) {
	slog.Debug("RealLabService.Status", "component", "web.services")
	state, err := s.Manager.Load()
	if err != nil {
		slog.Error("RealLabService.Status load failed", "component", "web.services", "error", err)
		return "", err
	}
	slog.Debug("RealLabService.Status loaded", "component", "web.services",
		"lab", state.Name, "nodes", len(state.Nodes))
	return "lab " + state.Name + " has nodes", nil
}

type RealDoctorService struct {
	Doctor interface {
		DiagnoseDNS(ctx context.Context, sessionID, controlPlane, question string) (string, error)
	}
	Audit AuditLogger
}

func (s RealDoctorService) DiagnoseDNS(r *http.Request, question string) (string, error) {
	start := time.Now()
	slog.Info("RealDoctorService.DiagnoseDNS start",
		"component", "web.services",
		"question_chars", len(question),
		"preview", logging.Truncate(question, 200))
	answer, err := s.Doctor.DiagnoseDNS(r.Context(), "web-session", "k8s-doc-lab-cp-1", question)
	dur := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("RealDoctorService.DiagnoseDNS failed",
			"component", "web.services", "duration_ms", dur, "error", err)
	} else {
		slog.Info("RealDoctorService.DiagnoseDNS ok",
			"component", "web.services", "duration_ms", dur, "answer_chars", len(answer))
	}
	if s.Audit != nil {
		status := "completed"
		if err != nil {
			status = "error: " + err.Error()
		}
		if auditErr := s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "diagnose_dns", Input: map[string]any{"question": question}, Result: status, DurationMS: dur}); auditErr != nil {
			slog.Warn("audit record failed", "component", "web.services", "tool", "diagnose_dns", "error", auditErr)
		}
	}
	return answer, err
}

func (s RealDoctorService) BreakDNS(r *http.Request) (string, error) {
	slog.Warn("RealDoctorService.BreakDNS not wired through playbook yet", "component", "web.services")
	return "DNS break action is wired through playbook service", nil
}
