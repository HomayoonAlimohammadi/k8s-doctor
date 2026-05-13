package web

import (
	"encoding/json"
	"net/http"
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
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/static/index.html")
	})
	s.mux.HandleFunc("/api/chat", s.handleChat)
	s.mux.HandleFunc("/api/lab/create", s.handleLabCreate)
	s.mux.HandleFunc("/api/lab/destroy", s.handleLabDestroy)
	s.mux.HandleFunc("/api/lab/status", s.handleLabStatus)
	s.mux.HandleFunc("/api/dns/break", s.handleDNSBreak)
}

func (s *Server) handleLabCreate(w http.ResponseWriter, r *http.Request) {
	if s.deps.Lab == nil {
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.CreateOrReuse(r)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleLabDestroy(w http.ResponseWriter, r *http.Request) {
	if s.deps.Lab == nil {
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.Destroy(r)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleLabStatus(w http.ResponseWriter, r *http.Request) {
	if s.deps.Lab == nil {
		writeJSON(w, map[string]string{"error": "lab service not configured"})
		return
	}
	msg, err := s.deps.Lab.Status(r)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleDNSBreak(w http.ResponseWriter, r *http.Request) {
	if s.deps.Doctor == nil {
		writeJSON(w, map[string]string{"error": "doctor service not configured"})
		return
	}
	msg, err := s.deps.Doctor.BreakDNS(r)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"status": msg})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	if s.deps.Doctor == nil {
		writeJSON(w, map[string]string{"answer": "Doctor service is not configured."})
		return
	}
	answer, err := s.deps.Doctor.DiagnoseDNS(r, payload.Question)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"answer": answer})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
