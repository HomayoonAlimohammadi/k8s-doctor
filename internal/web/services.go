package web

import (
	"context"
	"net/http"

	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/audit"
	"github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/lab"
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
	state, err := s.Manager.Create(r.Context(), lab.CreateOptions{ControlPlanes: 1})
	if err != nil {
		return "", err
	}
	msg := "lab ready: " + state.Name
	if s.Audit != nil {
		_ = s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "lab_create", Result: msg})
	}
	return msg, nil
}

func (s RealLabService) Destroy(r *http.Request) (string, error) {
	if err := s.Manager.Destroy(r.Context()); err != nil {
		return "", err
	}
	msg := "lab destroyed"
	if s.Audit != nil {
		_ = s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "lab_destroy", Result: msg})
	}
	return msg, nil
}

func (s RealLabService) Status(r *http.Request) (string, error) {
	state, err := s.Manager.Load()
	if err != nil {
		return "", err
	}
	return "lab " + state.Name + " has nodes", nil
}

type RealDoctorService struct {
	Doctor interface {
		DiagnoseDNS(ctx context.Context, sessionID, controlPlane, question string) (string, error)
	}
	Audit AuditLogger
}

func (s RealDoctorService) DiagnoseDNS(r *http.Request, question string) (string, error) {
	answer, err := s.Doctor.DiagnoseDNS(r.Context(), "web-session", "k8s-doc-lab-cp-1", question)
	if s.Audit != nil {
		status := "completed"
		if err != nil {
			status = "error: " + err.Error()
		}
		_ = s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "diagnose_dns", Input: map[string]any{"question": question}, Result: status})
	}
	return answer, err
}

func (s RealDoctorService) BreakDNS(r *http.Request) (string, error) {
	return "DNS break action is wired through playbook service", nil
}
