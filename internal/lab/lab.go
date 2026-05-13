package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Role string

const (
	RoleControlPlane Role = "control-plane"
	RoleWorker       Role = "worker"
)

type Node struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type State struct {
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
}

type Config struct {
	Name     string
	StateDir string
}

type CreateOptions struct {
	ControlPlanes int
	Workers       int
}

type Backend interface {
	Launch(ctx context.Context, name string) error
	Delete(ctx context.Context, name string) error
}

type Manager struct {
	backend Backend
	cfg     Config
}

func NewManager(backend Backend, cfg Config) *Manager {
	slog.Debug("lab.NewManager", "component", "lab", "name", cfg.Name, "state_dir", cfg.StateDir)
	return &Manager{backend: backend, cfg: cfg}
}

func (m *Manager) Create(ctx context.Context, opts CreateOptions) (State, error) {
	start := time.Now()
	if opts.ControlPlanes <= 0 {
		opts.ControlPlanes = 1
	}
	slog.Info("lab.Manager.Create start",
		"component", "lab", "lab", m.cfg.Name,
		"control_planes", opts.ControlPlanes, "workers", opts.Workers)
	state := State{Name: m.cfg.Name}
	for i := 1; i <= opts.ControlPlanes; i++ {
		name := fmt.Sprintf("%s-cp-%d", m.cfg.Name, i)
		slog.Info("lab.Manager.Create launching control-plane",
			"component", "lab", "node", name)
		if err := m.backend.Launch(ctx, name); err != nil {
			slog.Error("lab.Manager.Create launch control-plane failed",
				"component", "lab", "node", name, "error", err)
			return State{}, fmt.Errorf("launch control-plane node: %w", err)
		}
		state.Nodes = append(state.Nodes, Node{Name: name, Role: RoleControlPlane})
	}
	for i := 1; i <= opts.Workers; i++ {
		name := fmt.Sprintf("%s-worker-%d", m.cfg.Name, i)
		slog.Info("lab.Manager.Create launching worker",
			"component", "lab", "node", name)
		if err := m.backend.Launch(ctx, name); err != nil {
			slog.Error("lab.Manager.Create launch worker failed",
				"component", "lab", "node", name, "error", err)
			return State{}, fmt.Errorf("launch worker node: %w", err)
		}
		state.Nodes = append(state.Nodes, Node{Name: name, Role: RoleWorker})
	}
	if err := m.Save(state); err != nil {
		return State{}, err
	}
	slog.Info("lab.Manager.Create complete",
		"component", "lab", "lab", state.Name, "nodes", len(state.Nodes),
		"duration_ms", time.Since(start).Milliseconds())
	return state, nil
}

func (m *Manager) Save(state State) error {
	if err := os.MkdirAll(m.cfg.StateDir, 0o755); err != nil {
		slog.Error("lab.Manager.Save mkdir failed", "component", "lab", "dir", m.cfg.StateDir, "error", err)
		return fmt.Errorf("create state dir: %w", err)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		slog.Error("lab.Manager.Save marshal failed", "component", "lab", "error", err)
		return fmt.Errorf("marshal lab state: %w", err)
	}
	if err := os.WriteFile(m.statePath(), raw, 0o644); err != nil {
		slog.Error("lab.Manager.Save write failed", "component", "lab", "path", m.statePath(), "error", err)
		return fmt.Errorf("write lab state: %w", err)
	}
	slog.Debug("lab.Manager.Save ok", "component", "lab", "path", m.statePath(), "bytes", len(raw))
	return nil
}

func (m *Manager) Load() (State, error) {
	raw, err := os.ReadFile(m.statePath())
	if err != nil {
		slog.Debug("lab.Manager.Load read failed", "component", "lab", "path", m.statePath(), "error", err)
		return State{}, fmt.Errorf("read lab state: %w", err)
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		slog.Error("lab.Manager.Load parse failed", "component", "lab", "path", m.statePath(), "error", err)
		return State{}, fmt.Errorf("parse lab state: %w", err)
	}
	slog.Debug("lab.Manager.Load ok", "component", "lab", "lab", state.Name, "nodes", len(state.Nodes))
	return state, nil
}

func (m *Manager) Destroy(ctx context.Context) error {
	start := time.Now()
	slog.Info("lab.Manager.Destroy start", "component", "lab", "lab", m.cfg.Name)
	state, err := m.Load()
	if err != nil {
		slog.Warn("lab.Manager.Destroy load failed", "component", "lab", "error", err)
		return err
	}
	var firstErr error
	for _, node := range state.Nodes {
		slog.Info("lab.Manager.Destroy deleting node", "component", "lab", "node", node.Name)
		if err := m.backend.Delete(ctx, node.Name); err != nil {
			slog.Error("lab.Manager.Destroy delete failed",
				"component", "lab", "node", node.Name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if err := os.Remove(m.statePath()); err != nil && !os.IsNotExist(err) {
		slog.Warn("lab.Manager.Destroy remove state file failed",
			"component", "lab", "path", m.statePath(), "error", err)
		if firstErr == nil {
			firstErr = err
		}
	}
	slog.Info("lab.Manager.Destroy complete",
		"component", "lab", "lab", m.cfg.Name,
		"duration_ms", time.Since(start).Milliseconds(),
		"first_error", errString(firstErr))
	return firstErr
}

func (m *Manager) statePath() string {
	return filepath.Join(m.cfg.StateDir, "lab.json")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
