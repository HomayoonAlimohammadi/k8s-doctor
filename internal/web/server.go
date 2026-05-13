package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/HomayoonAlimohammadi/k8s-doctor/internal/logging"
)

type LabService interface {
	CreateOrReuse(r *http.Request) (string, error)
	Destroy(r *http.Request) (string, error)
	Status(r *http.Request) (string, error)
}

type DoctorService interface {
	DiagnoseDNS(r *http.Request, question string) (string, error)
	BreakDNS(r *http.Request) (string, error)
}

type Deps struct {
	Lab    LabService
	Doctor DoctorService
}

type Server struct {
	mux  *http.ServeMux
	deps Deps
}

func NewServer(deps Deps) *Server {
	s := &Server{mux: http.NewServeMux(), deps: deps}
	s.routes()
	slog.Debug("web server initialised",
		"component", "web.server", "lab_wired", deps.Lab != nil, "doctor_wired", deps.Doctor != nil)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	slog.Info("http request begin",
		"component", "web.server", "method", r.Method, "path", r.URL.Path,
		"remote", r.RemoteAddr, "user_agent", r.Header.Get("User-Agent"))
	s.mux.ServeHTTP(rec, r)
	slog.Info("http request end",
		"component", "web.server", "method", r.Method, "path", r.URL.Path,
		"status", rec.status, "bytes", rec.bytes,
		"duration_ms", time.Since(start).Milliseconds())
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("health check", "component", "web.server")
		writeJSON(w, map[string]string{"status": "ok"})
	})
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("serving index.html", "component", "web.server", "path", r.URL.Path)
		http.ServeFile(w, r, "web/static/index.html")
	})
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/lab/create", s.handleLabCreate)
	s.mux.HandleFunc("/api/lab/destroy", s.handleLabDestroy)
	s.mux.HandleFunc("/api/lab/status", s.handleLabStatus)
	s.mux.HandleFunc("/api/dns/break", s.handleDNSBreak)
}

func (s *Server) handleLabCreate(w http.ResponseWriter, r *http.Request) {
	slog.Info("handleLabCreate", "component", "web.server")
	if s.deps.Lab == nil {
		slog.Warn("lab service not configured", "component", "web.server", "handler", "lab.create")
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.CreateOrReuse(r)
	if err != nil {
		slog.Error("lab create failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("lab create ok", "component", "web.server", "message", msg)
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleLabDestroy(w http.ResponseWriter, r *http.Request) {
	slog.Info("handleLabDestroy", "component", "web.server")
	if s.deps.Lab == nil {
		slog.Warn("lab service not configured", "component", "web.server", "handler", "lab.destroy")
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.Destroy(r)
	if err != nil {
		slog.Error("lab destroy failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("lab destroy ok", "component", "web.server", "message", msg)
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleLabStatus(w http.ResponseWriter, r *http.Request) {
	slog.Info("handleLabStatus", "component", "web.server")
	if s.deps.Lab == nil {
		slog.Warn("lab service not configured", "component", "web.server", "handler", "lab.status")
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.Status(r)
	if err != nil {
		slog.Error("lab status failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("lab status ok", "component", "web.server", "message", msg)
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleDNSBreak(w http.ResponseWriter, r *http.Request) {
	slog.Info("handleDNSBreak", "component", "web.server")
	if s.deps.Doctor == nil {
		slog.Warn("doctor service not configured", "component", "web.server", "handler", "dns.break")
		writeJSON(w, map[string]string{"error": "doctor service not configured"})
		return
	}
	msg, err := s.deps.Doctor.BreakDNS(r)
	if err != nil {
		slog.Error("dns break failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("dns break ok", "component", "web.server", "message", msg)
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	slog.Info("handleChat", "component", "web.server")
	var payload struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("chat decode payload failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("chat question received",
		"component", "web.server",
		"chars", len(payload.Question),
		"preview", logging.Truncate(payload.Question, 200))
	if s.deps.Doctor == nil {
		slog.Warn("doctor service not configured", "component", "web.server", "handler", "chat")
		writeJSON(w, map[string]string{"answer": "Doctor service is not configured."})
		return
	}
	answer, err := s.deps.Doctor.DiagnoseDNS(r, payload.Question)
	if err != nil {
		slog.Error("chat diagnose failed", "component", "web.server", "error", err)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	slog.Info("chat answer ready",
		"component", "web.server", "chars", len(answer),
		"preview", logging.Truncate(answer, 200))
	writeJSON(w, map[string]string{"answer": answer})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("writeJSON encode failed", "component", "web.server", "error", err)
	}
}
