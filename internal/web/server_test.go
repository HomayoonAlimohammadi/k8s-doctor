package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	srv := NewServer(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestLabCreateUsesConfiguredService(t *testing.T) {
	srv := NewServer(Deps{Lab: fakeLabService{create: "lab ready"}})
	req := httptest.NewRequest(http.MethodPost, "/api/lab/create", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if !strings.Contains(rr.Body.String(), "lab ready") {
		t.Fatalf("response missing lab status: %s", rr.Body.String())
	}
}

func TestChatUsesDoctorService(t *testing.T) {
	srv := NewServer(Deps{Doctor: fakeDoctorService{answer: "CoreDNS diagnosis"}})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"question":"why is dns broken?"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)
	if !strings.Contains(rr.Body.String(), "CoreDNS diagnosis") {
		t.Fatalf("response missing diagnosis: %s", rr.Body.String())
	}
}

type fakeLabService struct {
	create  string
	destroy string
	status  string
	err     error
}

func (f fakeLabService) CreateOrReuse(r *http.Request) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.create, nil
}

func (f fakeLabService) Destroy(r *http.Request) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.destroy, nil
}

func (f fakeLabService) Status(r *http.Request) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.status, nil
}

type fakeDoctorService struct {
	answer string
	err    error
}

func (f fakeDoctorService) DiagnoseDNS(r *http.Request, question string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.answer, nil
}

func (f fakeDoctorService) BreakDNS(r *http.Request) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "dns broken", nil
}

var _ LabService = fakeLabService{}
var _ DoctorService = fakeDoctorService{}
var _ = errors.New
