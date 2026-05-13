package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	return &Manager{backend: backend, cfg: cfg}
}

func (m *Manager) Create(ctx context.Context, opts CreateOptions) (State, error) {
	if opts.ControlPlanes <= 0 {
		opts.ControlPlanes = 1
	}
	state := State{Name: m.cfg.Name}
	for i := 1; i <= opts.ControlPlanes; i++ {
		name := fmt.Sprintf("%s-cp-%d", m.cfg.Name, i)
		if err := m.backend.Launch(ctx, name); err != nil {
			return State{}, fmt.Errorf("launch control-plane node: %w", err)
		}
		state.Nodes = append(state.Nodes, Node{Name: name, Role: RoleControlPlane})
	}
	for i := 1; i <= opts.Workers; i++ {
		name := fmt.Sprintf("%s-worker-%d", m.cfg.Name, i)
		if err := m.backend.Launch(ctx, name); err != nil {
			return State{}, fmt.Errorf("launch worker node: %w", err)
		}
		state.Nodes = append(state.Nodes, Node{Name: name, Role: RoleWorker})
	}
	if err := m.Save(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (m *Manager) Save(state State) error {
	if err := os.MkdirAll(m.cfg.StateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lab state: %w", err)
	}
	if err := os.WriteFile(m.statePath(), raw, 0o644); err != nil {
		return fmt.Errorf("write lab state: %w", err)
	}
	return nil
}

func (m *Manager) Load() (State, error) {
	raw, err := os.ReadFile(m.statePath())
	if err != nil {
		return State{}, fmt.Errorf("read lab state: %w", err)
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("parse lab state: %w", err)
	}
	return state, nil
}

func (m *Manager) Destroy(ctx context.Context) error {
	state, err := m.Load()
	if err != nil {
		return err
	}
	var firstErr error
	for _, node := range state.Nodes {
		if err := m.backend.Delete(ctx, node.Name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := os.Remove(m.statePath()); err != nil && !os.IsNotExist(err) && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (m *Manager) statePath() string {
	return filepath.Join(m.cfg.StateDir, "lab.json")
}
