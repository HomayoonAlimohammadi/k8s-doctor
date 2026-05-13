# k8s-doc Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone Go-first web app under `hackathon/k8s-doc/` that answers Kubernetes/k8s-snap questions with docs citations, creates a disposable LXD k8s-snap lab, diagnoses a DNS/CoreDNS failure, applies a fix, verifies recovery, and records an audit trail.

**Architecture:** Approach B: one Go web/orchestrator app with a strong internal tool-runner boundary. The LLM sees high-level typed tools; low-level LXD, `k8s`, and `k8s kubectl` execution remains hidden behind deterministic adapters. MCP is not required for MVP, but every tool is shaped so an MCP adapter can be added later.

**Tech Stack:** Go, standard `net/http`, plain HTML/CSS/JavaScript, LXD CLI boundary, k8s snap `latest/stable`, OpenAI-compatible and Ollama-compatible HTTP providers, vector embeddings, local JSONL/file-backed state for MVP.

---

## Implementation rules

- Keep all project code under `hackathon/k8s-doc/`.
- Do not import internal packages from `src/k8s/pkg/...`.
- Do not integrate this project into snapcraft or k8s-snap CI unless explicitly requested later.
- Use typed interfaces and fakes for tests before real LXD integration.
- Use `context.Context` as the first parameter for all operations that may block or perform I/O.
- Wrap errors with `%w`.
- Never expose arbitrary shell execution to the model or web client.
- Real LXD tests must be opt-in via environment variables.
- Generated runtime state belongs under `hackathon/k8s-doc/.state/` and must be gitignored.

---

## Planned file structure

```text
hackathon/k8s-doc/
  .gitignore
  go.mod
  README.md
  cmd/k8s-doc/main.go
  internal/audit/audit.go
  internal/audit/audit_test.go
  internal/config/config.go
  internal/config/config_test.go
  internal/doctor/doctor.go
  internal/doctor/doctor_test.go
  internal/lab/lab.go
  internal/lab/lab_test.go
  internal/llm/llm.go
  internal/llm/openai.go
  internal/llm/ollama.go
  internal/llm/fake.go
  internal/playbooks/dns.go
  internal/playbooks/dns_test.go
  internal/rag/chunk.go
  internal/rag/chunk_test.go
  internal/rag/embedding.go
  internal/rag/index.go
  internal/rag/index_test.go
  internal/rag/source.go
  internal/rag/source_test.go
  internal/tools/registry.go
  internal/tools/registry_test.go
  internal/tools/runner.go
  internal/tools/runner_test.go
  internal/tools/k8ssnap/k8ssnap.go
  internal/tools/k8ssnap/k8ssnap_test.go
  internal/tools/kubectl/kubectl.go
  internal/tools/kubectl/kubectl_test.go
  internal/tools/lxd/lxd.go
  internal/tools/lxd/lxd_test.go
  internal/web/server.go
  internal/web/server_test.go
  web/static/index.html
  web/static/app.js
  web/static/styles.css
  docs/design.md
  docs/implementation-plan.md
```

### Responsibility map

| Path | Responsibility |
|------|----------------|
| `cmd/k8s-doc/main.go` | CLI entrypoint. Loads config, wires dependencies, starts web server, handles reindex command. |
| `internal/config` | Defaults, environment variables, config file loading, validation. |
| `internal/audit` | Append-only JSONL audit log for tool calls and underlying commands. |
| `internal/tools` | Tool interface, JSON schema metadata, registry, audited execution wrapper. |
| `internal/tools/lxd` | Constrained LXD operations and command construction. |
| `internal/tools/k8ssnap` | Node-local `k8s` operations through lab node execution. |
| `internal/tools/kubectl` | Node-local `k8s kubectl` operations through control-plane node execution. |
| `internal/lab` | Lab state, node inventory, create/reuse/destroy/reset flows. |
| `internal/rag` | Docs sources, chunking, embedding, vector index, retrieval and citations. |
| `internal/llm` | Chat and embedding provider abstractions plus HTTP providers. |
| `internal/playbooks` | DNS/CoreDNS diagnostic, break, repair, and verify workflows. |
| `internal/doctor` | Conversation orchestration and response formatting. |
| `internal/web` | HTTP handlers and static file serving. |
| `web/static` | Browser UI. |

---

## Phase 0: Project scaffold

### Task 0.1: Create standalone Go module

**Files:**
- Create: `hackathon/k8s-doc/go.mod`
- Create: `hackathon/k8s-doc/.gitignore`
- Create: `hackathon/k8s-doc/README.md`

- [ ] Create `go.mod`:

```go
module github.com/canonical/k8s-snap/hackathon/k8s-doc

go 1.24.13
```

- [ ] Create `.gitignore`:

```gitignore
.state/
bin/
coverage.out
*.test
```

- [ ] Create `README.md`:

```markdown
# k8s-doc

Hackathon prototype: Kubernetes Doctor for disposable k8s-snap labs.

## MVP

- Web chat UI
- k8s-snap and upstream Kubernetes docs retrieval
- LXD-backed local lab
- k8s snap install from `latest/stable`
- DNS/CoreDNS break/fix diagnosis
- Append-only audit log

## Run

```bash
go run ./cmd/k8s-doc serve
```

## Test

```bash
go test ./...
```
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: `go` reports no packages or all packages pass once later tasks add packages.

### Task 0.2: Add main command skeleton

**Files:**
- Create: `hackathon/k8s-doc/cmd/k8s-doc/main.go`

- [ ] Create command skeleton:

```go
package main

import (
    "context"
    "fmt"
    "os"
)

func main() {
    if err := run(context.Background(), os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "k8s-doc: %v\n", err)
        os.Exit(1)
    }
}

func run(ctx context.Context, args []string) error {
    command := "serve"
    if len(args) > 0 {
        command = args[0]
    }

    switch command {
    case "serve":
        fmt.Println("k8s-doc server wiring will be added in a later task")
        return nil
    case "reindex":
        fmt.Println("k8s-doc docs reindex wiring will be added in a later task")
        return nil
    default:
        return fmt.Errorf("unknown command %q", command)
    }
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

---

## Phase 1: Configuration

### Task 1.1: Implement config defaults and environment loading

**Files:**
- Create: `hackathon/k8s-doc/internal/config/config.go`
- Create: `hackathon/k8s-doc/internal/config/config_test.go`

- [ ] Write failing tests for defaults and env overrides:

```go
package config

import "testing"

func TestLoadDefaults(t *testing.T) {
    t.Setenv("K8S_DOC_LAB_NAME", "")
    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load returned error: %v", err)
    }
    if cfg.LabName != "k8s-doc-lab" {
        t.Fatalf("LabName = %q, want k8s-doc-lab", cfg.LabName)
    }
    if cfg.SnapChannel != "latest/stable" {
        t.Fatalf("SnapChannel = %q, want latest/stable", cfg.SnapChannel)
    }
    if cfg.StateDir != ".state" {
        t.Fatalf("StateDir = %q, want .state", cfg.StateDir)
    }
}

func TestLoadEnvironmentOverrides(t *testing.T) {
    t.Setenv("K8S_DOC_LAB_NAME", "demo")
    t.Setenv("K8S_DOC_LXD_REMOTE", "remote1")
    t.Setenv("K8S_DOC_LXD_IMAGE", "ubuntu:24.04")
    t.Setenv("K8S_DOC_SNAP_CHANNEL", "latest/edge")
    t.Setenv("K8S_DOC_STATE_DIR", "/tmp/k8s-doc-state")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load returned error: %v", err)
    }
    if cfg.LabName != "demo" || cfg.LXDRemote != "remote1" || cfg.LXDImage != "ubuntu:24.04" || cfg.SnapChannel != "latest/edge" || cfg.StateDir != "/tmp/k8s-doc-state" {
        t.Fatalf("unexpected config: %+v", cfg)
    }
}
```

- [ ] Run test to verify failure:

```bash
cd hackathon/k8s-doc && go test ./internal/config
```

Expected: FAIL because `Load` and `Config` do not exist.

- [ ] Implement config:

```go
package config

import "os"

type Config struct {
    LabName     string
    LXDRemote   string
    LXDImage    string
    SnapChannel string
    StateDir    string
    HTTPAddr    string

    ChatProvider      string
    ChatModel         string
    EmbeddingProvider string
    EmbeddingModel    string

    K8sSnapDocsPath string
    UpstreamDocsPath string
}

func Load() (Config, error) {
    return Config{
        LabName:           env("K8S_DOC_LAB_NAME", "k8s-doc-lab"),
        LXDRemote:         env("K8S_DOC_LXD_REMOTE", "local"),
        LXDImage:          env("K8S_DOC_LXD_IMAGE", "ubuntu:22.04"),
        SnapChannel:       env("K8S_DOC_SNAP_CHANNEL", "latest/stable"),
        StateDir:          env("K8S_DOC_STATE_DIR", ".state"),
        HTTPAddr:          env("K8S_DOC_HTTP_ADDR", "127.0.0.1:8080"),
        ChatProvider:      env("K8S_DOC_CHAT_PROVIDER", "openai"),
        ChatModel:         env("K8S_DOC_CHAT_MODEL", "gpt-4o-mini"),
        EmbeddingProvider: env("K8S_DOC_EMBEDDING_PROVIDER", "openai"),
        EmbeddingModel:    env("K8S_DOC_EMBEDDING_MODEL", "text-embedding-3-small"),
        K8sSnapDocsPath:   env("K8S_DOC_K8S_SNAP_DOCS", "../../docs/canonicalk8s"),
        UpstreamDocsPath:  env("K8S_DOC_UPSTREAM_K8S_DOCS", ".state/upstream-kubernetes-docs"),
    }, nil
}

func env(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/config
```

Expected: PASS.

---

## Phase 2: Audit logging

### Task 2.1: Implement append-only audit writer

**Files:**
- Create: `hackathon/k8s-doc/internal/audit/audit.go`
- Create: `hackathon/k8s-doc/internal/audit/audit_test.go`

- [ ] Write tests:

```go
package audit

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

func TestLoggerWritesJSONL(t *testing.T) {
    dir := t.TempDir()
    logger := NewLogger(filepath.Join(dir, "audit.jsonl"))

    entry := Entry{SessionID: "s1", Tool: "cluster_status", Input: map[string]any{"node": "cp1"}, Result: "ok"}
    if err := logger.Record(context.Background(), entry); err != nil {
        t.Fatalf("Record returned error: %v", err)
    }

    raw, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
    if err != nil {
        t.Fatalf("ReadFile returned error: %v", err)
    }

    var got Entry
    if err := json.Unmarshal(raw[:len(raw)-1], &got); err != nil {
        t.Fatalf("audit line is not JSON: %v", err)
    }
    if got.SessionID != "s1" || got.Tool != "cluster_status" || got.Result != "ok" || got.Timestamp.IsZero() {
        t.Fatalf("unexpected entry: %+v", got)
    }
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/audit
```

Expected: FAIL because audit package is missing.

- [ ] Implement audit logger:

```go
package audit

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type Entry struct {
    Timestamp  time.Time      `json:"timestamp"`
    SessionID  string         `json:"session_id"`
    Tool       string         `json:"tool"`
    Input      map[string]any `json:"input,omitempty"`
    Commands   []Command      `json:"commands,omitempty"`
    Result     string         `json:"result"`
    Error      string         `json:"error,omitempty"`
    DurationMS int64          `json:"duration_ms,omitempty"`
}

type Command struct {
    Args       []string `json:"args"`
    ExitCode   int      `json:"exit_code"`
    Stdout     string   `json:"stdout,omitempty"`
    Stderr     string   `json:"stderr,omitempty"`
    DurationMS int64    `json:"duration_ms,omitempty"`
}

type Logger struct {
    path string
    mu   sync.Mutex
}

func NewLogger(path string) *Logger {
    return &Logger{path: path}
}

func (l *Logger) Record(ctx context.Context, entry Entry) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    if entry.Timestamp.IsZero() {
        entry.Timestamp = time.Now().UTC()
    }

    if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
        return fmt.Errorf("create audit directory: %w", err)
    }

    f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return fmt.Errorf("open audit log: %w", err)
    }
    defer f.Close()

    enc := json.NewEncoder(f)
    if err := enc.Encode(entry); err != nil {
        return fmt.Errorf("write audit entry: %w", err)
    }
    return nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/audit
```

Expected: PASS.

---

## Phase 3: Command runner and typed tools

### Task 3.1: Implement command runner abstraction

**Files:**
- Create: `hackathon/k8s-doc/internal/tools/runner.go`
- Create: `hackathon/k8s-doc/internal/tools/runner_test.go`

- [ ] Write tests for fake runner and result shape:

```go
package tools

import (
    "context"
    "testing"
)

func TestFakeRunnerRecordsCommands(t *testing.T) {
    runner := NewFakeRunner(CommandResult{Stdout: "ok\n", ExitCode: 0})
    result, err := runner.Run(context.Background(), []string{"echo", "ok"}, RunOptions{})
    if err != nil {
        t.Fatalf("Run returned error: %v", err)
    }
    if result.Stdout != "ok\n" || result.ExitCode != 0 {
        t.Fatalf("unexpected result: %+v", result)
    }
    if len(runner.Commands()) != 1 || runner.Commands()[0][0] != "echo" {
        t.Fatalf("commands not recorded: %+v", runner.Commands())
    }
}
```

- [ ] Implement runner types:

```go
package tools

import (
    "bytes"
    "context"
    "fmt"
    "os/exec"
    "sync"
    "time"
)

type RunOptions struct {
    Input   []byte
    WorkDir string
}

type CommandResult struct {
    Stdout     string
    Stderr     string
    ExitCode   int
    DurationMS int64
}

type Runner interface {
    Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error) {
    if len(args) == 0 {
        return CommandResult{ExitCode: -1}, fmt.Errorf("command args are empty")
    }
    start := time.Now()
    cmd := exec.CommandContext(ctx, args[0], args[1:]...)
    cmd.Dir = opts.WorkDir
    cmd.Stdin = bytes.NewReader(opts.Input)
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    err := cmd.Run()
    exitCode := 0
    if err != nil {
        exitCode = 1
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        }
    }
    result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode, DurationMS: time.Since(start).Milliseconds()}
    if err != nil {
        return result, fmt.Errorf("run %q: %w", args[0], err)
    }
    return result, nil
}

type FakeRunner struct {
    result   CommandResult
    err      error
    commands [][]string
    mu       sync.Mutex
}

func NewFakeRunner(result CommandResult) *FakeRunner {
    return &FakeRunner{result: result}
}

func NewFailingFakeRunner(result CommandResult, err error) *FakeRunner {
    return &FakeRunner{result: result, err: err}
}

func (f *FakeRunner) Run(ctx context.Context, args []string, opts RunOptions) (CommandResult, error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    copied := append([]string(nil), args...)
    f.commands = append(f.commands, copied)
    return f.result, f.err
}

func (f *FakeRunner) Commands() [][]string {
    f.mu.Lock()
    defer f.mu.Unlock()
    copied := make([][]string, len(f.commands))
    for i := range f.commands {
        copied[i] = append([]string(nil), f.commands[i]...)
    }
    return copied
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools
```

Expected: PASS.

### Task 3.2: Implement tool interface and audited registry

**Files:**
- Create: `hackathon/k8s-doc/internal/tools/registry.go`
- Create: `hackathon/k8s-doc/internal/tools/registry_test.go`

- [ ] Write tests:

```go
package tools

import (
    "context"
    "encoding/json"
    "testing"
)

type echoTool struct{}

func (echoTool) Name() string { return "echo" }
func (echoTool) Description() string { return "echo test tool" }
func (echoTool) InputSchema() JSONSchema { return JSONSchema{Type: "object"} }
func (echoTool) Execute(ctx context.Context, input json.RawMessage) (ToolResult, error) {
    return ToolResult{Summary: string(input), Data: map[string]any{"ok": true}}, nil
}

func TestRegistryExecutesTool(t *testing.T) {
    r := NewRegistry()
    if err := r.Register(echoTool{}); err != nil {
        t.Fatalf("Register returned error: %v", err)
    }
    result, err := r.Execute(context.Background(), "echo", []byte(`{"message":"hi"}`))
    if err != nil {
        t.Fatalf("Execute returned error: %v", err)
    }
    if result.Summary != `{"message":"hi"}` {
        t.Fatalf("unexpected summary: %q", result.Summary)
    }
}
```

- [ ] Implement registry:

```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "sort"
)

type JSONSchema struct {
    Type       string                `json:"type"`
    Properties map[string]JSONSchema `json:"properties,omitempty"`
    Required   []string              `json:"required,omitempty"`
    Enum       []string              `json:"enum,omitempty"`
    Items      *JSONSchema           `json:"items,omitempty"`
}

type ToolResult struct {
    Summary  string         `json:"summary"`
    Data     map[string]any `json:"data,omitempty"`
    Commands []CommandResult `json:"commands,omitempty"`
}

type Tool interface {
    Name() string
    Description() string
    InputSchema() JSONSchema
    Execute(ctx context.Context, input json.RawMessage) (ToolResult, error)
}

type Registry struct {
    tools map[string]Tool
}

func NewRegistry() *Registry {
    return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(tool Tool) error {
    if tool.Name() == "" {
        return fmt.Errorf("tool name is empty")
    }
    if _, exists := r.tools[tool.Name()]; exists {
        return fmt.Errorf("tool %q already registered", tool.Name())
    }
    r.tools[tool.Name()] = tool
    return nil
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (ToolResult, error) {
    tool, ok := r.tools[name]
    if !ok {
        return ToolResult{}, fmt.Errorf("unknown tool %q", name)
    }
    return tool.Execute(ctx, input)
}

func (r *Registry) Names() []string {
    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools
```

Expected: PASS.

---

## Phase 4: LXD tool package

### Task 4.1: Implement constrained LXD client

**Files:**
- Create: `hackathon/k8s-doc/internal/tools/lxd/lxd.go`
- Create: `hackathon/k8s-doc/internal/tools/lxd/lxd_test.go`

- [ ] Write tests for command construction:

```go
package lxd

import (
    "context"
    "reflect"
    "testing"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

func TestLaunchUsesProfilesAndImage(t *testing.T) {
    runner := tools.NewFakeRunner(tools.CommandResult{Stdout: "", ExitCode: 0})
    client := NewClient(runner, Config{Remote: "local", Image: "ubuntu:22.04", Profile: "default"})
    if err := client.Launch(context.Background(), "k8s-doc-lab-cp1"); err != nil {
        t.Fatalf("Launch returned error: %v", err)
    }
    got := runner.Commands()[0]
    want := []string{"lxc", "launch", "ubuntu:22.04", "k8s-doc-lab-cp1", "-p", "default"}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("command = %#v, want %#v", got, want)
    }
}

func TestExecRejectsEmptyCommand(t *testing.T) {
    runner := tools.NewFakeRunner(tools.CommandResult{})
    client := NewClient(runner, Config{Image: "ubuntu:22.04", Profile: "default"})
    if _, err := client.Exec(context.Background(), "node1", nil); err == nil {
        t.Fatal("Exec returned nil error for empty command")
    }
}
```

- [ ] Implement LXD client:

```go
package lxd

import (
    "context"
    "fmt"
    "regexp"
    "strings"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type Config struct {
    Remote  string
    Image   string
    Profile string
}

type Client struct {
    runner tools.Runner
    cfg    Config
}

var safeInstanceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,62}$`)

func NewClient(runner tools.Runner, cfg Config) *Client {
    if cfg.Profile == "" {
        cfg.Profile = "default"
    }
    return &Client{runner: runner, cfg: cfg}
}

func (c *Client) Launch(ctx context.Context, name string) error {
    if err := validateInstanceName(name); err != nil {
        return err
    }
    image := c.cfg.Image
    if image == "" {
        image = "ubuntu:22.04"
    }
    args := []string{"lxc", "launch", image, name, "-p", c.cfg.Profile}
    _, err := c.runner.Run(ctx, args, tools.RunOptions{})
    if err != nil {
        return fmt.Errorf("launch LXD instance %q: %w", name, err)
    }
    return nil
}

func (c *Client) Delete(ctx context.Context, name string) error {
    if err := validateInstanceName(name); err != nil {
        return err
    }
    _, err := c.runner.Run(ctx, []string{"lxc", "rm", name, "--force"}, tools.RunOptions{})
    if err != nil {
        return fmt.Errorf("delete LXD instance %q: %w", name, err)
    }
    return nil
}

func (c *Client) Exec(ctx context.Context, name string, command []string) (tools.CommandResult, error) {
    if err := validateInstanceName(name); err != nil {
        return tools.CommandResult{}, err
    }
    if len(command) == 0 {
        return tools.CommandResult{}, fmt.Errorf("exec command is empty")
    }
    for _, part := range command {
        if strings.TrimSpace(part) == "" {
            return tools.CommandResult{}, fmt.Errorf("exec command contains empty argument")
        }
    }
    args := append([]string{"lxc", "exec", name, "--"}, command...)
    result, err := c.runner.Run(ctx, args, tools.RunOptions{})
    if err != nil {
        return result, fmt.Errorf("exec in LXD instance %q: %w", name, err)
    }
    return result, nil
}

func validateInstanceName(name string) error {
    if !safeInstanceName.MatchString(name) {
        return fmt.Errorf("unsafe LXD instance name %q", name)
    }
    return nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools/lxd
```

Expected: PASS.

---

## Phase 5: Lab lifecycle

### Task 5.1: Implement lab state and create/reuse/destroy flow with fake LXD

**Files:**
- Create: `hackathon/k8s-doc/internal/lab/lab.go`
- Create: `hackathon/k8s-doc/internal/lab/lab_test.go`

- [ ] Write tests:

```go
package lab

import (
    "context"
    "testing"
)

type fakeBackend struct { created []string; deleted []string }
func (f *fakeBackend) Launch(ctx context.Context, name string) error { f.created = append(f.created, name); return nil }
func (f *fakeBackend) Delete(ctx context.Context, name string) error { f.deleted = append(f.deleted, name); return nil }

func TestCreateDefaultLab(t *testing.T) {
    backend := &fakeBackend{}
    manager := NewManager(backend, Config{Name: "demo", StateDir: t.TempDir()})
    state, err := manager.Create(context.Background(), CreateOptions{ControlPlanes: 1})
    if err != nil { t.Fatalf("Create returned error: %v", err) }
    if len(state.Nodes) != 1 || state.Nodes[0].Name != "demo-cp-1" || state.Nodes[0].Role != RoleControlPlane {
        t.Fatalf("unexpected state: %+v", state)
    }
    if len(backend.created) != 1 || backend.created[0] != "demo-cp-1" {
        t.Fatalf("unexpected created instances: %+v", backend.created)
    }
}
```

- [ ] Implement lab manager:

```go
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
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/lab
```

Expected: PASS.

---

## Phase 6: k8s-snap and kubectl tool packages

### Task 6.1: Implement k8s-snap node commands

**Files:**
- Create: `hackathon/k8s-doc/internal/tools/k8ssnap/k8ssnap.go`
- Create: `hackathon/k8s-doc/internal/tools/k8ssnap/k8ssnap_test.go`

- [ ] Write tests for command shapes:

```go
package k8ssnap

import (
    "context"
    "reflect"
    "testing"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type fakeNodeRunner struct { commands [][]string }
func (f *fakeNodeRunner) Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
    f.commands = append(f.commands, append([]string{node}, command...))
    return tools.CommandResult{Stdout: "ok", ExitCode: 0}, nil
}

func TestBootstrapCommand(t *testing.T) {
    runner := &fakeNodeRunner{}
    client := NewClient(runner, Config{SnapChannel: "latest/stable"})
    if _, err := client.Bootstrap(context.Background(), "cp1"); err != nil { t.Fatalf("Bootstrap error: %v", err) }
    want := []string{"cp1", "sudo", "k8s", "bootstrap"}
    if !reflect.DeepEqual(runner.commands[0], want) { t.Fatalf("command = %#v, want %#v", runner.commands[0], want) }
}
```

- [ ] Implement k8s-snap client:

```go
package k8ssnap

import (
    "context"
    "fmt"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type NodeRunner interface {
    Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error)
}

type Config struct { SnapChannel string }

type Client struct { runner NodeRunner; cfg Config }

func NewClient(runner NodeRunner, cfg Config) *Client { return &Client{runner: runner, cfg: cfg} }

func (c *Client) Install(ctx context.Context, node string) (tools.CommandResult, error) {
    channel := c.cfg.SnapChannel
    if channel == "" { channel = "latest/stable" }
    return c.exec(ctx, node, []string{"sudo", "snap", "install", "k8s", "--classic", "--channel", channel})
}

func (c *Client) Bootstrap(ctx context.Context, node string) (tools.CommandResult, error) {
    return c.exec(ctx, node, []string{"sudo", "k8s", "bootstrap"})
}

func (c *Client) Status(ctx context.Context, node string) (tools.CommandResult, error) {
    return c.exec(ctx, node, []string{"sudo", "k8s", "status"})
}

func (c *Client) exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
    result, err := c.runner.Exec(ctx, node, command)
    if err != nil { return result, fmt.Errorf("run k8s-snap command on %s: %w", node, err) }
    return result, nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools/k8ssnap
```

Expected: PASS.

### Task 6.2: Implement constrained kubectl operations

**Files:**
- Create: `hackathon/k8s-doc/internal/tools/kubectl/kubectl.go`
- Create: `hackathon/k8s-doc/internal/tools/kubectl/kubectl_test.go`

- [ ] Write tests for allowlisted commands:

```go
package kubectl

import (
    "context"
    "reflect"
    "testing"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type fakeNodeRunner struct { commands [][]string }
func (f *fakeNodeRunner) Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) {
    f.commands = append(f.commands, append([]string{node}, command...))
    return tools.CommandResult{Stdout: "ok", ExitCode: 0}, nil
}

func TestGetPodsAllNamespaces(t *testing.T) {
    runner := &fakeNodeRunner{}
    client := NewClient(runner)
    if _, err := client.Get(context.Background(), "cp1", "pods", "", true); err != nil { t.Fatalf("Get error: %v", err) }
    want := []string{"cp1", "sudo", "k8s", "kubectl", "get", "pods", "-A", "-o", "wide"}
    if !reflect.DeepEqual(runner.commands[0], want) { t.Fatalf("command = %#v, want %#v", runner.commands[0], want) }
}
```

- [ ] Implement kubectl client:

```go
package kubectl

import (
    "context"
    "fmt"
    "regexp"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type NodeRunner interface { Exec(ctx context.Context, node string, command []string) (tools.CommandResult, error) }

type Client struct { runner NodeRunner }

var safeName = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

func NewClient(runner NodeRunner) *Client { return &Client{runner: runner} }

func (c *Client) Get(ctx context.Context, node, resource, namespace string, allNamespaces bool) (tools.CommandResult, error) {
    if !safeName.MatchString(resource) { return tools.CommandResult{}, fmt.Errorf("unsafe resource %q", resource) }
    args := []string{"sudo", "k8s", "kubectl", "get", resource}
    if allNamespaces { args = append(args, "-A") }
    if namespace != "" {
        if !safeName.MatchString(namespace) { return tools.CommandResult{}, fmt.Errorf("unsafe namespace %q", namespace) }
        args = append(args, "-n", namespace)
    }
    args = append(args, "-o", "wide")
    return c.runner.Exec(ctx, node, args)
}

func (c *Client) Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error) {
    for label, value := range map[string]string{"resource": resource, "name": name, "namespace": namespace} {
        if value != "" && !safeName.MatchString(value) { return tools.CommandResult{}, fmt.Errorf("unsafe %s %q", label, value) }
    }
    args := []string{"sudo", "k8s", "kubectl", "describe", resource, name}
    if namespace != "" { args = append(args, "-n", namespace) }
    return c.runner.Exec(ctx, node, args)
}

func (c *Client) Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error) {
    if !safeName.MatchString(pod) || !safeName.MatchString(namespace) { return tools.CommandResult{}, fmt.Errorf("unsafe pod or namespace") }
    if tail <= 0 { tail = 100 }
    return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "logs", pod, "-n", namespace, "--tail", fmt.Sprint(tail)})
}

func (c *Client) ApplyYAML(ctx context.Context, node string, yamlPath string) (tools.CommandResult, error) {
    if !safeName.MatchString(yamlPath) { return tools.CommandResult{}, fmt.Errorf("unsafe yaml path %q", yamlPath) }
    return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "apply", "-f", yamlPath})
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools/kubectl
```

Expected: PASS.

---

## Phase 7: RAG sources, chunking, embeddings, and vector index

### Task 7.1: Implement document source and markdown loader

**Files:**
- Create: `hackathon/k8s-doc/internal/rag/source.go`
- Create: `hackathon/k8s-doc/internal/rag/source_test.go`

- [ ] Write tests:

```go
package rag

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestDirectorySourceLoadsMarkdown(t *testing.T) {
    dir := t.TempDir()
    if err := os.WriteFile(filepath.Join(dir, "dns.md"), []byte("# DNS\nCoreDNS docs"), 0o644); err != nil { t.Fatal(err) }
    docs, err := NewDirectorySource("k8s-snap", dir).Load(context.Background())
    if err != nil { t.Fatalf("Load error: %v", err) }
    if len(docs) != 1 || docs[0].Source != "k8s-snap" || docs[0].Text != "# DNS\nCoreDNS docs" {
        t.Fatalf("unexpected docs: %+v", docs)
    }
}
```

- [ ] Implement source loader:

```go
package rag

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type Document struct {
    Source string
    Path   string
    Text   string
}

type Source interface { Load(ctx context.Context) ([]Document, error) }

type DirectorySource struct { name string; root string }

func NewDirectorySource(name, root string) DirectorySource { return DirectorySource{name: name, root: root} }

func (s DirectorySource) Load(ctx context.Context) ([]Document, error) {
    var docs []Document
    err := filepath.WalkDir(s.root, func(path string, d os.DirEntry, err error) error {
        if err != nil { return err }
        select { case <-ctx.Done(): return ctx.Err(); default: }
        if d.IsDir() { return nil }
        if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".markdown") { return nil }
        raw, err := os.ReadFile(path)
        if err != nil { return fmt.Errorf("read doc %s: %w", path, err) }
        docs = append(docs, Document{Source: s.name, Path: path, Text: string(raw)})
        return nil
    })
    if err != nil { return nil, fmt.Errorf("load directory source %s: %w", s.root, err) }
    return docs, nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/rag
```

Expected: PASS.

### Task 7.2: Implement markdown chunker

**Files:**
- Create/modify: `hackathon/k8s-doc/internal/rag/chunk.go`
- Create/modify: `hackathon/k8s-doc/internal/rag/chunk_test.go`

- [ ] Write tests:

```go
package rag

import "testing"

func TestChunkDocumentIncludesCitationFields(t *testing.T) {
    doc := Document{Source: "k8s-snap", Path: "dns.md", Text: "# DNS\nCoreDNS config\n\n## Troubleshooting\nCheck pods"}
    chunks := ChunkDocument(doc, 40)
    if len(chunks) == 0 { t.Fatal("expected chunks") }
    if chunks[0].Source != "k8s-snap" || chunks[0].Path != "dns.md" || chunks[0].Text == "" {
        t.Fatalf("unexpected chunk: %+v", chunks[0])
    }
}
```

- [ ] Implement chunking:

```go
package rag

import "strings"

type Chunk struct {
    ID      string  `json:"id"`
    Source  string  `json:"source"`
    Path    string  `json:"path"`
    Heading string  `json:"heading,omitempty"`
    Text    string  `json:"text"`
    Vector  []float64 `json:"vector,omitempty"`
}

func ChunkDocument(doc Document, maxChars int) []Chunk {
    if maxChars <= 0 { maxChars = 1200 }
    sections := splitByHeading(doc.Text)
    var chunks []Chunk
    for _, section := range sections {
        text := strings.TrimSpace(section.text)
        if text == "" { continue }
        for len(text) > maxChars {
            chunks = append(chunks, newChunk(doc, section.heading, text[:maxChars], len(chunks)))
            text = strings.TrimSpace(text[maxChars:])
        }
        chunks = append(chunks, newChunk(doc, section.heading, text, len(chunks)))
    }
    return chunks
}

type section struct { heading string; text string }

func splitByHeading(text string) []section {
    lines := strings.Split(text, "\n")
    current := section{}
    sections := []section{}
    for _, line := range lines {
        if strings.HasPrefix(line, "#") {
            if strings.TrimSpace(current.text) != "" { sections = append(sections, current) }
            current = section{heading: strings.TrimSpace(strings.TrimLeft(line, "# ")), text: line + "\n"}
            continue
        }
        current.text += line + "\n"
    }
    if strings.TrimSpace(current.text) != "" { sections = append(sections, current) }
    return sections
}

func newChunk(doc Document, heading string, text string, index int) Chunk {
    return Chunk{ID: doc.Source + ":" + doc.Path + ":" + string(rune(index+'0')), Source: doc.Source, Path: doc.Path, Heading: heading, Text: strings.TrimSpace(text)}
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/rag
```

Expected: PASS.

### Task 7.3: Implement embedder interface and local vector index

**Files:**
- Create: `hackathon/k8s-doc/internal/rag/embedding.go`
- Create: `hackathon/k8s-doc/internal/rag/index.go`
- Create: `hackathon/k8s-doc/internal/rag/index_test.go`

- [ ] Write tests with deterministic fake embeddings:

```go
package rag

import (
    "context"
    "testing"
)

type fakeEmbedder struct{}
func (fakeEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
    vectors := make([][]float64, len(texts))
    for i, text := range texts {
        if len(text) > 0 && (text[0] == 'd' || text[0] == 'D') { vectors[i] = []float64{1, 0} } else { vectors[i] = []float64{0, 1} }
    }
    return vectors, nil
}

func TestIndexSearchReturnsNearestChunk(t *testing.T) {
    idx := NewMemoryIndex(fakeEmbedder{})
    chunks := []Chunk{{ID: "dns", Text: "dns troubleshooting"}, {ID: "storage", Text: "storage troubleshooting"}}
    if err := idx.Add(context.Background(), chunks); err != nil { t.Fatalf("Add error: %v", err) }
    hits, err := idx.Search(context.Background(), "dns", 1)
    if err != nil { t.Fatalf("Search error: %v", err) }
    if len(hits) != 1 || hits[0].Chunk.ID != "dns" { t.Fatalf("unexpected hits: %+v", hits) }
}
```

- [ ] Implement embedder and index:

```go
package rag

import "context"

type Embedder interface { Embed(ctx context.Context, texts []string) ([][]float64, error) }
```

```go
package rag

import (
    "context"
    "fmt"
    "math"
    "sort"
)

type SearchHit struct {
    Chunk Chunk
    Score float64
}

type MemoryIndex struct { embedder Embedder; chunks []Chunk }

func NewMemoryIndex(embedder Embedder) *MemoryIndex { return &MemoryIndex{embedder: embedder} }

func (i *MemoryIndex) Add(ctx context.Context, chunks []Chunk) error {
    texts := make([]string, len(chunks))
    for n, chunk := range chunks { texts[n] = chunk.Text }
    vectors, err := i.embedder.Embed(ctx, texts)
    if err != nil { return fmt.Errorf("embed chunks: %w", err) }
    for n := range chunks { chunks[n].Vector = vectors[n] }
    i.chunks = append(i.chunks, chunks...)
    return nil
}

func (i *MemoryIndex) Search(ctx context.Context, query string, limit int) ([]SearchHit, error) {
    if limit <= 0 { limit = 5 }
    vectors, err := i.embedder.Embed(ctx, []string{query})
    if err != nil { return nil, fmt.Errorf("embed query: %w", err) }
    queryVector := vectors[0]
    hits := make([]SearchHit, 0, len(i.chunks))
    for _, chunk := range i.chunks {
        hits = append(hits, SearchHit{Chunk: chunk, Score: cosine(queryVector, chunk.Vector)})
    }
    sort.Slice(hits, func(a, b int) bool { return hits[a].Score > hits[b].Score })
    if len(hits) > limit { hits = hits[:limit] }
    return hits, nil
}

func cosine(a, b []float64) float64 {
    if len(a) != len(b) || len(a) == 0 { return 0 }
    var dot, aa, bb float64
    for idx := range a { dot += a[idx]*b[idx]; aa += a[idx]*a[idx]; bb += b[idx]*b[idx] }
    if aa == 0 || bb == 0 { return 0 }
    return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/rag
```

Expected: PASS.

---

## Phase 8: LLM providers

### Task 8.1: Implement provider interfaces and fake model

**Files:**
- Create: `hackathon/k8s-doc/internal/llm/llm.go`
- Create: `hackathon/k8s-doc/internal/llm/fake.go`

- [ ] Implement interfaces:

```go
package llm

import "context"

type Message struct { Role string `json:"role"`; Content string `json:"content"` }

type ChatRequest struct { Messages []Message `json:"messages"`; Tools []ToolSpec `json:"tools,omitempty"` }

type ToolSpec struct { Name string `json:"name"`; Description string `json:"description"`; Schema any `json:"schema"` }

type ChatResponse struct { Content string `json:"content"` }

type ChatModel interface { Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) }

type EmbeddingModel interface { Embed(ctx context.Context, texts []string) ([][]float64, error) }
```

- [ ] Implement fake model:

```go
package llm

import "context"

type FakeChatModel struct { Response string }
func (f FakeChatModel) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) { return ChatResponse{Content: f.Response}, nil }

type FakeEmbeddingModel struct{}
func (FakeEmbeddingModel) Embed(ctx context.Context, texts []string) ([][]float64, error) {
    vectors := make([][]float64, len(texts))
    for i, text := range texts { vectors[i] = []float64{float64(len(text)), 1} }
    return vectors, nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/llm
```

Expected: PASS.

### Task 8.2: Add OpenAI-compatible and Ollama-compatible provider stubs

**Files:**
- Create: `hackathon/k8s-doc/internal/llm/openai.go`
- Create: `hackathon/k8s-doc/internal/llm/ollama.go`

- [ ] Implement OpenAI-compatible client with explicit unsupported tool-call behavior for MVP:

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type OpenAIClient struct { BaseURL string; APIKey string; Model string; HTTPClient *http.Client }

func (c OpenAIClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    body := map[string]any{"model": c.Model, "messages": req.Messages}
    raw, err := json.Marshal(body)
    if err != nil { return ChatResponse{}, fmt.Errorf("marshal openai request: %w", err) }
    httpClient := c.HTTPClient
    if httpClient == nil { httpClient = http.DefaultClient }
    endpoint := c.BaseURL
    if endpoint == "" { endpoint = "https://api.openai.com/v1/chat/completions" }
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
    if err != nil { return ChatResponse{}, fmt.Errorf("create openai request: %w", err) }
    httpReq.Header.Set("Content-Type", "application/json")
    if c.APIKey != "" { httpReq.Header.Set("Authorization", "Bearer "+c.APIKey) }
    resp, err := httpClient.Do(httpReq)
    if err != nil { return ChatResponse{}, fmt.Errorf("call openai-compatible API: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 { return ChatResponse{}, fmt.Errorf("openai-compatible API status %d", resp.StatusCode) }
    var decoded struct { Choices []struct { Message Message `json:"message"` } `json:"choices"` }
    if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil { return ChatResponse{}, fmt.Errorf("decode openai response: %w", err) }
    if len(decoded.Choices) == 0 { return ChatResponse{}, fmt.Errorf("openai response contained no choices") }
    return ChatResponse{Content: decoded.Choices[0].Message.Content}, nil
}
```

- [ ] Implement Ollama client:

```go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type OllamaClient struct { BaseURL string; Model string; HTTPClient *http.Client }

func (c OllamaClient) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
    prompt := ""
    for _, msg := range req.Messages { prompt += msg.Role + ": " + msg.Content + "\n" }
    body := map[string]any{"model": c.Model, "prompt": prompt, "stream": false}
    raw, err := json.Marshal(body)
    if err != nil { return ChatResponse{}, fmt.Errorf("marshal ollama request: %w", err) }
    httpClient := c.HTTPClient
    if httpClient == nil { httpClient = http.DefaultClient }
    endpoint := c.BaseURL
    if endpoint == "" { endpoint = "http://127.0.0.1:11434/api/generate" }
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
    if err != nil { return ChatResponse{}, fmt.Errorf("create ollama request: %w", err) }
    httpReq.Header.Set("Content-Type", "application/json")
    resp, err := httpClient.Do(httpReq)
    if err != nil { return ChatResponse{}, fmt.Errorf("call ollama API: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 { return ChatResponse{}, fmt.Errorf("ollama API status %d", resp.StatusCode) }
    var decoded struct { Response string `json:"response"` }
    if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil { return ChatResponse{}, fmt.Errorf("decode ollama response: %w", err) }
    return ChatResponse{Content: decoded.Response}, nil
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/llm
```

Expected: PASS.

---

## Phase 9: DNS playbook

### Task 9.1: Implement DNS diagnostic data collection playbook

**Files:**
- Create: `hackathon/k8s-doc/internal/playbooks/dns.go`
- Create: `hackathon/k8s-doc/internal/playbooks/dns_test.go`

- [ ] Write tests:

```go
package playbooks

import (
    "context"
    "testing"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type fakeKubectl struct{ calls []string }
func (f *fakeKubectl) Get(ctx context.Context, node, resource, namespace string, all bool) (tools.CommandResult, error) { f.calls = append(f.calls, "get "+resource); return tools.CommandResult{Stdout: "ok"}, nil }
func (f *fakeKubectl) Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error) { f.calls = append(f.calls, "describe "+resource); return tools.CommandResult{Stdout: "ok"}, nil }
func (f *fakeKubectl) Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error) { f.calls = append(f.calls, "logs "+pod); return tools.CommandResult{Stdout: "ok"}, nil }
func (f *fakeKubectl) ApplyYAML(ctx context.Context, node, yamlPath string) (tools.CommandResult, error) { f.calls = append(f.calls, "apply "+yamlPath); return tools.CommandResult{Stdout: "ok"}, nil }

func TestDNSCollectCallsExpectedResources(t *testing.T) {
    kube := &fakeKubectl{}
    pb := NewDNSPlaybook(kube)
    report, err := pb.Collect(context.Background(), "cp1")
    if err != nil { t.Fatalf("Collect error: %v", err) }
    if report.Summary == "" || len(kube.calls) < 4 { t.Fatalf("unexpected report/calls: %+v %#v", report, kube.calls) }
}
```

- [ ] Implement DNS playbook:

```go
package playbooks

import (
    "context"
    "fmt"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type Kubectl interface {
    Get(ctx context.Context, node, resource, namespace string, allNamespaces bool) (tools.CommandResult, error)
    Describe(ctx context.Context, node, resource, name, namespace string) (tools.CommandResult, error)
    Logs(ctx context.Context, node, pod, namespace string, tail int) (tools.CommandResult, error)
    ApplyYAML(ctx context.Context, node, yamlPath string) (tools.CommandResult, error)
}

type DNSReport struct {
    Summary string
    Evidence map[string]string
}

type DNSPlaybook struct { kubectl Kubectl }

func NewDNSPlaybook(kubectl Kubectl) *DNSPlaybook { return &DNSPlaybook{kubectl: kubectl} }

func (p *DNSPlaybook) Collect(ctx context.Context, controlPlane string) (DNSReport, error) {
    evidence := map[string]string{}
    calls := []struct{ key string; fn func() (tools.CommandResult, error) }{
        {"pods", func() (tools.CommandResult, error) { return p.kubectl.Get(ctx, controlPlane, "pods", "kube-system", false) }},
        {"svc", func() (tools.CommandResult, error) { return p.kubectl.Get(ctx, controlPlane, "svc/kube-dns", "kube-system", false) }},
        {"endpoints", func() (tools.CommandResult, error) { return p.kubectl.Get(ctx, controlPlane, "endpoints/kube-dns", "kube-system", false) }},
        {"configmap", func() (tools.CommandResult, error) { return p.kubectl.Describe(ctx, controlPlane, "configmap", "coredns", "kube-system") }},
    }
    for _, call := range calls {
        result, err := call.fn()
        if err != nil { return DNSReport{}, fmt.Errorf("collect DNS %s: %w", call.key, err) }
        evidence[call.key] = result.Stdout
    }
    return DNSReport{Summary: "Collected CoreDNS pods, service, endpoints, and ConfigMap evidence.", Evidence: evidence}, nil
}

func (p *DNSPlaybook) BreakByScalingToZero(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
    return p.kubectl.ApplyYAML(ctx, controlPlane, "/tmp/k8s-doc-break-dns.yaml")
}

func (p *DNSPlaybook) RepairByScalingToOne(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
    return p.kubectl.ApplyYAML(ctx, controlPlane, "/tmp/k8s-doc-repair-dns.yaml")
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/playbooks
```

Expected: PASS.

---

## Phase 10: Doctor orchestration

### Task 10.1: Implement answer formatting with citations, evidence, and audit summary

**Files:**
- Create: `hackathon/k8s-doc/internal/doctor/doctor.go`
- Create: `hackathon/k8s-doc/internal/doctor/doctor_test.go`

- [ ] Write tests:

```go
package doctor

import "testing"

func TestFormatAnswerIncludesRequiredSections(t *testing.T) {
    answer := FormatAnswer(Answer{
        Summary: "DNS is broken because CoreDNS has no running pods.",
        Diagnosis: "CoreDNS deployment was scaled to zero.",
        Evidence: []string{"coredns replicas: 0"},
        Fix: "Scaled CoreDNS back to one replica.",
        Verification: "DNS probe resolved kubernetes.default.",
        Citations: []Citation{{Source: "kubernetes", Path: "dns.md", Snippet: "DNS service discovery"}},
        ToolsRun: []string{"dns_collect", "dns_repair", "dns_verify"},
    })
    for _, section := range []string{"Summary", "Diagnosis", "Evidence", "Fix", "Verification", "Docs references", "Tools run"} {
        if !contains(answer, section) { t.Fatalf("answer missing section %q:\n%s", section, answer) }
    }
}

func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || contains(s[1:], sub) || s[:len(sub)] == sub)) }
```

- [ ] Implement formatter:

```go
package doctor

import "strings"

type Citation struct { Source string; Path string; Snippet string }

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
    for _, item := range answer.Evidence { b.WriteString("- " + item + "\n") }
    b.WriteString("\n## Fix\n\n" + answer.Fix + "\n\n")
    b.WriteString("## Verification\n\n" + answer.Verification + "\n\n")
    b.WriteString("## Docs references\n\n")
    for _, citation := range answer.Citations { b.WriteString("- " + citation.Source + ": `" + citation.Path + "` — " + citation.Snippet + "\n") }
    b.WriteString("\n## Tools run\n\n")
    for _, tool := range answer.ToolsRun { b.WriteString("- " + tool + "\n") }
    return b.String()
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/doctor
```

Expected: PASS.

### Task 10.2: Implement DNS diagnosis orchestration with fakes

**Files:**
- Modify: `hackathon/k8s-doc/internal/doctor/doctor.go`
- Modify: `hackathon/k8s-doc/internal/doctor/doctor_test.go`

- [ ] Add test:

```go
func TestDoctorDiagnosesDNSWithDocsAndPlaybook(t *testing.T) {
    d := Doctor{
        Retriever: FakeRetriever{Hits: []Citation{{Source: "kubernetes", Path: "dns.md", Snippet: "CoreDNS provides cluster DNS."}}},
        DNS: FakeDNS{Report: DNSReport{Summary: "CoreDNS pods are unavailable.", Evidence: []string{"no coredns pods"}}},
    }
    answer, err := d.DiagnoseDNS(context.Background(), "session1", "cp1", "Why is DNS broken?")
    if err != nil { t.Fatalf("DiagnoseDNS error: %v", err) }
    if !contains(answer, "CoreDNS") || !contains(answer, "Docs references") { t.Fatalf("unexpected answer:\n%s", answer) }
}
```

- [ ] Add orchestration types:

```go
import "context"

type Retriever interface { Search(ctx context.Context, query string, limit int) ([]Citation, error) }

type DNSDiagnostic interface { Collect(ctx context.Context, controlPlane string) (DNSReport, error) }

type DNSReport struct { Summary string; Evidence []string }

type Doctor struct { Retriever Retriever; DNS DNSDiagnostic }

func (d Doctor) DiagnoseDNS(ctx context.Context, sessionID, controlPlane, question string) (string, error) {
    citations, err := d.Retriever.Search(ctx, question, 5)
    if err != nil { return "", fmt.Errorf("search docs: %w", err) }
    report, err := d.DNS.Collect(ctx, controlPlane)
    if err != nil { return "", fmt.Errorf("collect DNS diagnostics: %w", err) }
    return FormatAnswer(Answer{
        Summary: "DNS appears unhealthy based on live CoreDNS evidence.",
        Diagnosis: report.Summary,
        Evidence: report.Evidence,
        Fix: "For the MVP DNS scenario, restore CoreDNS to a healthy replica count and re-run the DNS probe.",
        Verification: "Run dns_verify after repair to confirm kubernetes.default resolves from a test pod.",
        Citations: citations,
        ToolsRun: []string{"docs_search", "dns_collect"},
    }), nil
}

type FakeRetriever struct { Hits []Citation }
func (f FakeRetriever) Search(ctx context.Context, query string, limit int) ([]Citation, error) { return f.Hits, nil }

type FakeDNS struct { Report DNSReport }
func (f FakeDNS) Collect(ctx context.Context, controlPlane string) (DNSReport, error) { return f.Report, nil }
```

- [ ] Fix imports and run:

```bash
cd hackathon/k8s-doc && go test ./internal/doctor
```

Expected: PASS.

---

## Phase 11: Web server and UI

### Task 11.1: Implement HTTP API skeleton

**Files:**
- Create: `hackathon/k8s-doc/internal/web/server.go`
- Create: `hackathon/k8s-doc/internal/web/server_test.go`

- [ ] Write tests:

```go
package web

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthEndpoint(t *testing.T) {
    srv := NewServer(Deps{})
    req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
    rr := httptest.NewRecorder()
    srv.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK { t.Fatalf("status = %d, want 200", rr.Code) }
}
```

- [ ] Implement server:

```go
package web

import (
    "encoding/json"
    "net/http"
)

type Deps struct{}

type Server struct { mux *http.ServeMux; deps Deps }

func NewServer(deps Deps) *Server {
    s := &Server{mux: http.NewServeMux(), deps: deps}
    s.routes()
    return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
    s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, map[string]string{"status": "ok"})
    })
}

func writeJSON(w http.ResponseWriter, value any) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(value)
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/web
```

Expected: PASS.

### Task 11.2: Add static web UI

**Files:**
- Create: `hackathon/k8s-doc/web/static/index.html`
- Create: `hackathon/k8s-doc/web/static/app.js`
- Create: `hackathon/k8s-doc/web/static/styles.css`
- Modify: `hackathon/k8s-doc/internal/web/server.go`

- [ ] Create HTML:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>k8s-doc</title>
  <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
  <aside>
    <h1>k8s-doc</h1>
    <button id="create-lab">Create/reuse lab</button>
    <button id="break-dns">Break DNS</button>
    <button id="diagnose-dns">Diagnose DNS</button>
    <button id="destroy-lab">Destroy lab</button>
    <pre id="lab-status">Lab status: unknown</pre>
  </aside>
  <main>
    <section id="messages"></section>
    <form id="chat-form">
      <input id="question" placeholder="Ask why something is broken..." autocomplete="off">
      <button type="submit">Ask</button>
    </form>
  </main>
  <script src="/static/app.js"></script>
</body>
</html>
```

- [ ] Create JavaScript:

```javascript
const messages = document.querySelector('#messages');
const form = document.querySelector('#chat-form');
const question = document.querySelector('#question');

function addMessage(role, text) {
  const article = document.createElement('article');
  article.className = role;
  article.textContent = text;
  messages.appendChild(article);
  messages.scrollTop = messages.scrollHeight;
}

form.addEventListener('submit', async (event) => {
  event.preventDefault();
  const text = question.value.trim();
  if (!text) return;
  question.value = '';
  addMessage('user', text);
  const response = await fetch('/api/chat', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({question: text})});
  const payload = await response.json();
  addMessage('doctor', payload.answer || payload.error || 'No answer');
});
```

- [ ] Create CSS:

```css
body { margin: 0; display: grid; grid-template-columns: 280px 1fr; min-height: 100vh; font-family: system-ui, sans-serif; background: #111827; color: #f9fafb; }
aside { padding: 1rem; background: #1f2937; border-right: 1px solid #374151; }
main { display: grid; grid-template-rows: 1fr auto; }
button, input { font: inherit; padding: .6rem; border-radius: .4rem; border: 1px solid #4b5563; }
button { display: block; width: 100%; margin: .5rem 0; background: #2563eb; color: white; cursor: pointer; }
#messages { padding: 1rem; overflow: auto; }
article { white-space: pre-wrap; margin: .75rem 0; padding: .75rem; border-radius: .5rem; }
.user { background: #374151; }
.doctor { background: #064e3b; }
form { display: grid; grid-template-columns: 1fr auto; gap: .5rem; padding: 1rem; background: #1f2937; }
```

- [ ] Modify server to serve static files and chat placeholder:

```go
s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "web/static/index.html") })
s.mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"answer": "Doctor orchestration will be connected in the next task."}) })
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/web
```

Expected: PASS.

### Task 11.3: Wire `cmd/k8s-doc serve` to web server

**Files:**
- Modify: `hackathon/k8s-doc/cmd/k8s-doc/main.go`

- [ ] Update `serve` case to load config and listen:

```go
cfg, err := config.Load()
if err != nil { return err }
server := web.NewServer(web.Deps{})
fmt.Printf("k8s-doc listening on http://%s\n", cfg.HTTPAddr)
return http.ListenAndServe(cfg.HTTPAddr, server)
```

- [ ] Add imports:

```go
import (
    "context"
    "fmt"
    "net/http"
    "os"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/config"
    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/web"
)
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

- [ ] Manual smoke test:

```bash
cd hackathon/k8s-doc && go run ./cmd/k8s-doc serve
```

Expected: server prints `k8s-doc listening on http://127.0.0.1:8080` and web UI loads.

---

## Phase 12: Tool registration and web API actions

### Task 12.1: Add web handlers for lab and DNS actions using interfaces

**Files:**
- Modify: `hackathon/k8s-doc/internal/web/server.go`
- Modify: `hackathon/k8s-doc/internal/web/server_test.go`

- [ ] Define interfaces in `web/server.go`:

```go
type LabService interface {
    CreateOrReuse(r *http.Request) (string, error)
    Destroy(r *http.Request) (string, error)
    Status(r *http.Request) (string, error)
}

type DoctorService interface {
    DiagnoseDNS(r *http.Request, question string) (string, error)
    BreakDNS(r *http.Request) (string, error)
}

type Deps struct { Lab LabService; Doctor DoctorService }
```

- [ ] Add endpoints:

```go
s.mux.HandleFunc("/api/lab/create", s.handleLabCreate)
s.mux.HandleFunc("/api/lab/destroy", s.handleLabDestroy)
s.mux.HandleFunc("/api/lab/status", s.handleLabStatus)
s.mux.HandleFunc("/api/dns/break", s.handleDNSBreak)
```

- [ ] Add handler pattern:

```go
func (s *Server) handleLabCreate(w http.ResponseWriter, r *http.Request) {
    if s.deps.Lab == nil { writeJSON(w, map[string]string{"error": "lab service not configured"}); return }
    msg, err := s.deps.Lab.CreateOrReuse(r)
    if err != nil { writeJSON(w, map[string]string{"error": err.Error()}); return }
    writeJSON(w, map[string]string{"status": msg})
}
```

- [ ] Make `/api/chat` call `Doctor.DiagnoseDNS` when question contains `dns`:

```go
var payload struct { Question string `json:"question"` }
if err := json.NewDecoder(r.Body).Decode(&payload); err != nil { writeJSON(w, map[string]string{"error": err.Error()}); return }
if s.deps.Doctor == nil { writeJSON(w, map[string]string{"answer": "Doctor service is not configured."}); return }
answer, err := s.deps.Doctor.DiagnoseDNS(r, payload.Question)
if err != nil { writeJSON(w, map[string]string{"error": err.Error()}); return }
writeJSON(w, map[string]string{"answer": answer})
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/web
```

Expected: PASS.

---

## Phase 13: Real lab wiring

### Task 13.1: Connect LXD backend to lab manager

**Files:**
- Modify: `hackathon/k8s-doc/cmd/k8s-doc/main.go`

- [ ] Create an adapter from `lxd.Client` to `lab.Backend`:

```go
type lxdBackend struct { client *lxd.Client }
func (b lxdBackend) Launch(ctx context.Context, name string) error { return b.client.Launch(ctx, name) }
func (b lxdBackend) Delete(ctx context.Context, name string) error { return b.client.Delete(ctx, name) }
```

- [ ] Wire runner, LXD client, and lab manager in serve command:

```go
runner := tools.ExecRunner{}
lxdClient := lxd.NewClient(runner, lxd.Config{Remote: cfg.LXDRemote, Image: cfg.LXDImage, Profile: "default"})
labManager := lab.NewManager(lxdBackend{client: lxdClient}, lab.Config{Name: cfg.LabName, StateDir: cfg.StateDir})
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

### Task 13.2: Implement lab service for web handlers

**Files:**
- Create: `hackathon/k8s-doc/internal/web/services.go`
- Create: `hackathon/k8s-doc/internal/web/services_test.go`

- [ ] Implement `LabService` backed by lab manager:

```go
package web

import (
    "context"
    "net/http"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/lab"
)

type LabManager interface {
    Create(ctx context.Context, opts lab.CreateOptions) (lab.State, error)
    Destroy(ctx context.Context) error
    Load() (lab.State, error)
}

type RealLabService struct { Manager LabManager }

func (s RealLabService) CreateOrReuse(r *http.Request) (string, error) {
    state, err := s.Manager.Create(r.Context(), lab.CreateOptions{ControlPlanes: 1})
    if err != nil { return "", err }
    return "lab ready: " + state.Name, nil
}

func (s RealLabService) Destroy(r *http.Request) (string, error) {
    if err := s.Manager.Destroy(r.Context()); err != nil { return "", err }
    return "lab destroyed", nil
}

func (s RealLabService) Status(r *http.Request) (string, error) {
    state, err := s.Manager.Load()
    if err != nil { return "", err }
    return "lab " + state.Name + " has nodes", nil
}
```

- [ ] Wire it into server deps in `main.go`:

```go
server := web.NewServer(web.Deps{Lab: web.RealLabService{Manager: labManager}})
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

---

## Phase 14: Real cluster bootstrap smoke path

### Task 14.1: Add cluster bootstrap service

**Files:**
- Create: `hackathon/k8s-doc/internal/lab/cluster.go`
- Create: `hackathon/k8s-doc/internal/lab/cluster_test.go`

- [ ] Implement service using k8ssnap client:

```go
package lab

import (
    "context"
    "fmt"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
)

type K8sSnap interface {
    Install(ctx context.Context, node string) (tools.CommandResult, error)
    Bootstrap(ctx context.Context, node string) (tools.CommandResult, error)
    Status(ctx context.Context, node string) (tools.CommandResult, error)
}

type ClusterService struct { K8s K8sSnap }

func (s ClusterService) Bootstrap(ctx context.Context, state State) error {
    cp, ok := FirstControlPlane(state)
    if !ok { return fmt.Errorf("state has no control-plane node") }
    if _, err := s.K8s.Install(ctx, cp.Name); err != nil { return err }
    if _, err := s.K8s.Bootstrap(ctx, cp.Name); err != nil { return err }
    if _, err := s.K8s.Status(ctx, cp.Name); err != nil { return err }
    return nil
}

func FirstControlPlane(state State) (Node, bool) {
    for _, node := range state.Nodes { if node.Role == RoleControlPlane { return node, true } }
    return Node{}, false
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/lab
```

Expected: PASS.

---

## Phase 15: DNS break/fix real command manifests

### Task 15.1: Generate break and repair manifests on node

**Files:**
- Modify: `hackathon/k8s-doc/internal/playbooks/dns.go`
- Modify: `hackathon/k8s-doc/internal/playbooks/dns_test.go`

- [ ] Replace `BreakByScalingToZero` and `RepairByScalingToOne` implementation with direct `kubectl scale` support by extending kubectl client with `Scale`:

```go
func (c *Client) Scale(ctx context.Context, node, resource, namespace string, replicas int) (tools.CommandResult, error) {
    if !safeName.MatchString(resource) || !safeName.MatchString(namespace) { return tools.CommandResult{}, fmt.Errorf("unsafe scale target") }
    return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "scale", resource, "-n", namespace, "--replicas", fmt.Sprint(replicas)})
}
```

- [ ] Extend playbook interface:

```go
Scale(ctx context.Context, node, resource, namespace string, replicas int) (tools.CommandResult, error)
```

- [ ] Implement:

```go
func (p *DNSPlaybook) BreakByScalingToZero(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
    return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 0)
}

func (p *DNSPlaybook) RepairByScalingToOne(ctx context.Context, controlPlane string) (tools.CommandResult, error) {
    return p.kubectl.Scale(ctx, controlPlane, "deployment/coredns", "kube-system", 1)
}
```

- [ ] Add verify method using a temporary pod command through kubectl run:

```go
func (c *Client) RunDNSProbe(ctx context.Context, node string) (tools.CommandResult, error) {
    return c.runner.Exec(ctx, node, []string{"sudo", "k8s", "kubectl", "run", "k8s-doc-dns-probe", "--image=busybox:1.36", "--restart=Never", "--rm", "-i", "--", "nslookup", "kubernetes.default"})
}
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./internal/tools/kubectl ./internal/playbooks
```

Expected: PASS.

---

## Phase 16: End-to-end web orchestration

### Task 16.1: Wire doctor DNS flow into web chat

**Files:**
- Modify: `hackathon/k8s-doc/cmd/k8s-doc/main.go`
- Modify: `hackathon/k8s-doc/internal/web/services.go`

- [ ] Create `RealDoctorService` that calls doctor orchestration:

```go
type RealDoctorService struct { Doctor interface { DiagnoseDNS(ctx context.Context, sessionID, controlPlane, question string) (string, error) } }

func (s RealDoctorService) DiagnoseDNS(r *http.Request, question string) (string, error) {
    return s.Doctor.DiagnoseDNS(r.Context(), "web-session", "k8s-doc-lab-cp-1", question)
}

func (s RealDoctorService) BreakDNS(r *http.Request) (string, error) {
    return "DNS break action is wired through playbook service", nil
}
```

- [ ] Wire dependencies in `main.go` using fake retriever first, then replace with real RAG after Phase 17:

```go
doc := doctor.Doctor{
    Retriever: doctor.FakeRetriever{Hits: []doctor.Citation{{Source: "k8s-snap", Path: "docs/canonicalk8s", Snippet: "k8s-snap DNS documentation"}}},
    DNS: doctor.FakeDNS{Report: doctor.DNSReport{Summary: "CoreDNS evidence collected", Evidence: []string{"CoreDNS check executed"}}},
}
server := web.NewServer(web.Deps{Lab: web.RealLabService{Manager: labManager}, Doctor: web.RealDoctorService{Doctor: doc}})
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

---

## Phase 17: Real RAG indexing command

### Task 17.1: Implement `k8s-doc reindex`

**Files:**
- Modify: `hackathon/k8s-doc/cmd/k8s-doc/main.go`
- Create: `hackathon/k8s-doc/internal/rag/reindex.go`
- Create: `hackathon/k8s-doc/internal/rag/reindex_test.go`

- [ ] Implement reindex service:

```go
package rag

import (
    "context"
    "fmt"
)

type Reindexer struct { Sources []Source; Index *MemoryIndex; MaxChunkChars int }

func (r Reindexer) Reindex(ctx context.Context) (int, error) {
    total := 0
    for _, source := range r.Sources {
        docs, err := source.Load(ctx)
        if err != nil { return total, fmt.Errorf("load source: %w", err) }
        for _, doc := range docs {
            chunks := ChunkDocument(doc, r.MaxChunkChars)
            if err := r.Index.Add(ctx, chunks); err != nil { return total, err }
            total += len(chunks)
        }
    }
    return total, nil
}
```

- [ ] Wire command:

```go
case "reindex":
    cfg, err := config.Load()
    if err != nil { return err }
    index := rag.NewMemoryIndex(llm.FakeEmbeddingModel{})
    count, err := rag.Reindexer{Sources: []rag.Source{rag.NewDirectorySource("k8s-snap", cfg.K8sSnapDocsPath), rag.NewDirectorySource("upstream-kubernetes", cfg.UpstreamDocsPath)}, Index: index, MaxChunkChars: 1200}.Reindex(ctx)
    if err != nil { return err }
    fmt.Printf("indexed %d chunks\n", count)
    return nil
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

---

## Phase 18: Audit integration

### Task 18.1: Record every web-triggered operation

**Files:**
- Modify: `hackathon/k8s-doc/internal/web/services.go`
- Modify: `hackathon/k8s-doc/cmd/k8s-doc/main.go`

- [ ] Add logger to `RealLabService` and `RealDoctorService`:

```go
type AuditLogger interface { Record(ctx context.Context, entry audit.Entry) error }
```

- [ ] For `CreateOrReuse`, record:

```go
_ = s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "lab_create", Result: msg})
```

- [ ] For `DiagnoseDNS`, record:

```go
_ = s.Audit.Record(r.Context(), audit.Entry{SessionID: "web-session", Tool: "diagnose_dns", Input: map[string]any{"question": question}, Result: "completed"})
```

- [ ] Wire logger in `main.go`:

```go
auditLogger := audit.NewLogger(filepath.Join(cfg.StateDir, "audit.jsonl"))
```

- [ ] Run:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS.

---

## Phase 19: Real LXD smoke test script and opt-in integration test

### Task 19.1: Add opt-in real LXD smoke test

**Files:**
- Create: `hackathon/k8s-doc/internal/lab/lxd_integration_test.go`

- [ ] Add build-safe, env-gated test:

```go
package lab

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools"
    "github.com/canonical/k8s-snap/hackathon/k8s-doc/internal/tools/lxd"
)

func TestRealLXDCreateDestroy(t *testing.T) {
    if os.Getenv("K8S_DOC_RUN_LXD_TESTS") != "1" { t.Skip("set K8S_DOC_RUN_LXD_TESTS=1 to run real LXD tests") }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    client := lxd.NewClient(tools.ExecRunner{}, lxd.Config{Image: "ubuntu:22.04", Profile: "default"})
    backend := lxdBackendForTest{client: client}
    manager := NewManager(backend, Config{Name: "k8s-doc-test", StateDir: t.TempDir()})
    if _, err := manager.Create(ctx, CreateOptions{ControlPlanes: 1}); err != nil { t.Fatalf("Create error: %v", err) }
    if err := manager.Destroy(ctx); err != nil { t.Fatalf("Destroy error: %v", err) }
}

type lxdBackendForTest struct { client *lxd.Client }
func (b lxdBackendForTest) Launch(ctx context.Context, name string) error { return b.client.Launch(ctx, name) }
func (b lxdBackendForTest) Delete(ctx context.Context, name string) error { return b.client.Delete(ctx, name) }
```

- [ ] Run default tests:

```bash
cd hackathon/k8s-doc && go test ./...
```

Expected: PASS with LXD test skipped.

- [ ] Run opt-in real test only when LXD is configured:

```bash
cd hackathon/k8s-doc && K8S_DOC_RUN_LXD_TESTS=1 go test ./internal/lab -run TestRealLXDCreateDestroy -v
```

Expected: PASS on a configured LXD host.

---

## Phase 20: Demo hardening

### Task 20.1: Add demo checklist markdown

**Files:**
- Create: `hackathon/k8s-doc/docs/demo.md`

- [ ] Create demo runbook:

```markdown
# k8s-doc Demo Runbook

## Prerequisites

- LXD installed and initialized
- Host can run `lxc launch ubuntu:22.04 demo-test`
- Network allows snap install from `latest/stable`
- LLM provider configured
- Upstream Kubernetes docs source configured

## Start

```bash
cd hackathon/k8s-doc
go run ./cmd/k8s-doc reindex
go run ./cmd/k8s-doc serve
```

Open http://127.0.0.1:8080.

## Demo path

1. Create/reuse lab.
2. Bootstrap cluster.
3. Ask: "Why is DNS broken?"
4. Break DNS.
5. Diagnose DNS.
6. Repair DNS.
7. Verify DNS resolution.
8. Open `.state/audit.jsonl` to show tool trace.

## Recovery

```bash
lxc rm k8s-doc-lab-cp-1 --force
rm -rf .state
```
```

- [ ] Run markdown sanity by reading file manually.

### Task 20.2: Final verification checklist

**Files:**
- Modify: `hackathon/k8s-doc/docs/demo.md`

- [ ] Add acceptance checklist:

```markdown
## Acceptance checklist

- [ ] `go test ./...` passes.
- [ ] `go run ./cmd/k8s-doc serve` starts web UI.
- [ ] `go run ./cmd/k8s-doc reindex` indexes docs.
- [ ] Web UI can create or reuse a lab.
- [ ] Lab can install k8s from `latest/stable`.
- [ ] Lab can run `k8s status`.
- [ ] Lab can run `k8s kubectl get nodes`.
- [ ] DNS break action changes cluster state.
- [ ] DNS diagnosis shows evidence and docs citations.
- [ ] DNS repair restores healthy resolution.
- [ ] `.state/audit.jsonl` contains tool calls and command summaries.
```

---

## Phase 21: Completion criteria

The project is complete when all of these are true:

- [ ] All non-LXD tests pass with `cd hackathon/k8s-doc && go test ./...`.
- [ ] Web UI starts with `go run ./cmd/k8s-doc serve`.
- [ ] Docs can be indexed from configured k8s-snap and upstream Kubernetes docs paths.
- [ ] User can ask a docs-backed question and receive a cited answer.
- [ ] User can create or reuse a local LXD lab.
- [ ] Lab installs `k8s` from `latest/stable`.
- [ ] Lab bootstraps a one-node control-plane cluster.
- [ ] Tooling can run `k8s status` on the node.
- [ ] Tooling can run `k8s kubectl get nodes` on the node.
- [ ] DNS break scenario is deterministic.
- [ ] DNS diagnosis collects live evidence.
- [ ] DNS repair applies a state-changing fix.
- [ ] DNS verification proves recovery.
- [ ] Every user-triggered operation writes an audit entry.
- [ ] Demo runbook can be followed from a fresh checkout.

---

## Recommended execution order

1. Phases 0-3: Build testable skeleton and tool abstractions.
2. Phases 4-6: Add LXD/k8s/kubectl command construction with fakes.
3. Phase 7: Add RAG indexing and retrieval.
4. Phase 8: Add LLM provider abstraction.
5. Phases 9-10: Add DNS playbook and doctor orchestration.
6. Phase 11: Add web UI and API.
7. Phases 12-18: Wire real services and audit.
8. Phase 19: Run opt-in real LXD smoke tests.
9. Phase 20: Harden demo runbook.

## Risk register

| Risk | Mitigation |
|------|------------|
| LXD setup varies by host | Keep real LXD tests opt-in and provide cleanup commands. |
| Snap install is slow or network-dependent | Use persistent lab by default for demos. |
| Vector embeddings slow implementation | Keep embedder interface; start with fake/local deterministic embeddings in tests and wire provider later. |
| LLM tool calling adds complexity | Use deterministic orchestration for MVP and allow LLM to synthesize final answer. |
| DNS scenario too complex | Start by scaling CoreDNS deployment to zero; add ConfigMap corruption only as stretch. |
| Web UI scope creep | Use plain HTML/CSS/JS and minimal endpoints. |

## Stretch tasks after MVP

- [ ] Add MCP adapter around `internal/tools.Registry`.
- [ ] Add LoadBalancer/MetalLB diagnostic playbook.
- [ ] Add Gateway/Ingress diagnostic playbook.
- [ ] Add pod scheduling diagnostic playbook.
- [ ] Add report export as markdown.
- [ ] Add configurable docs source management in the UI.
- [ ] Add persistent vector index on disk instead of memory-only index.
- [ ] Add approval/read-only policy mode for future non-disposable clusters.
