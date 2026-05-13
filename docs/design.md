# k8s-doc Design

## Working name

`k8s-doc` — Kubernetes Doctor.

## One-sentence pitch

`k8s-doc` is a Go-first, web-based Kubernetes doctor that combines docs retrieval with live inspection and mutation of a disposable k8s-snap lab cluster to diagnose, fix, and verify Kubernetes problems.

## Context

This is a hackathon project kept under `hackathon/k8s-doc/` inside the `k8s-snap` checkout, but it is intentionally standalone. It treats k8s-snap as the distribution under test, not as an imported Go library.

The MVP targets a local or locally configured LXD environment and installs the `k8s` snap from `latest/stable`. Remote user clusters are explicitly out of scope for the MVP.

## Goals

1. Provide a browser-based chat experience for Kubernetes/k8s-snap troubleshooting.
2. Retrieve and cite relevant documentation from k8s-snap and upstream Kubernetes documentation sources.
3. Create and manage a disposable local k8s-snap lab on LXD.
4. Run live cluster checks through constrained typed tools rather than arbitrary shell access.
5. Diagnose one real end-to-end issue, apply a fix inside the disposable lab, and verify recovery.
6. Keep the architecture compatible with a future MCP adapter without requiring MCP for the MVP.
7. Keep every tool invocation and command in an audit log.

## Non-goals for MVP

1. Connecting to a user's production or remote Kubernetes cluster.
2. Running tools with unrestricted shell access exposed to the model.
3. Supporting every Kubernetes diagnostic domain deeply.
4. Building separate LXD and k8s-snap MCP servers before the core demo works.
5. Importing internal k8s-snap Go packages from `src/k8s`.
6. Integrating this project into k8s-snap snapcraft, CI, or release workflows.

## Chosen architecture: Approach B

The MVP uses a Go web application plus a strong internal tool-runner boundary.

One process can host the web UI and orchestrator, but code is separated into focused packages:

- `web`: HTTP server, static UI, chat endpoints.
- `doctor`: orchestration loop from user question to retrieval, tool execution, diagnosis, action, and final response.
- `rag`: documentation ingestion, chunking, embeddings, vector search, and source citation.
- `lab`: lab lifecycle, node inventory, cluster state, and cleanup.
- `tools`: typed tools, schemas, audit wrapper, and execution policy.
- `tools/lxd`: constrained LXD operations.
- `tools/k8ssnap`: constrained node-local `k8s` operations.
- `tools/kubectl`: constrained node-local `k8s kubectl` operations.
- `playbooks`: deterministic diagnostic and repair playbooks, starting with DNS/CoreDNS.
- `audit`: append-only logs for every tool call and underlying command.
- `llm`: model-provider abstraction for OpenAI-compatible HTTP APIs and Ollama.

The LLM should primarily see high-level tools. Low-level LXD and command execution are implementation details hidden behind deterministic adapters.

## Project location

```text
hackathon/k8s-doc/
  cmd/k8s-doc/
  internal/audit/
  internal/config/
  internal/doctor/
  internal/lab/
  internal/llm/
  internal/playbooks/
  internal/rag/
  internal/tools/
    k8ssnap/
    kubectl/
    lxd/
  web/
    static/
  docs/
    design.md
    implementation-plan.md
```

## System boundary decisions

### k8s-snap boundary

`k8s-doc` interacts with k8s-snap through the same external surfaces users and tests use:

- `snap install k8s --channel latest/stable --classic`
- node-local `k8s status`
- node-local `k8s bootstrap`
- node-local `k8s kubectl ...`
- LXD file copy and command execution

It does not import `src/k8s/pkg/...` packages. This keeps the hackathon project independent and avoids coupling to internal product APIs.

### LXD boundary

The lab layer owns instance creation and deletion. The tool layer exposes constrained operations such as:

- create lab
- list lab nodes
- run allowlisted node command
- push file
- pull diagnostic file
- delete lab

The LLM never receives `lxc shell` or arbitrary shell access directly.

### Kubernetes command boundary

Because k8s-snap commands are node-local, the MVP runs `k8s` and `k8s kubectl` inside LXD nodes. The user-facing abstraction is not `lxc exec`; it is typed operations such as:

- `cluster_status`
- `bootstrap_cluster`
- `kubectl_get`
- `kubectl_describe`
- `kubectl_logs`
- `collect_dns_diagnostics`
- `repair_dns`
- `verify_dns_resolution`

This hides the implementation detail while preserving k8s-snap behavior.

## MVP demo story

Primary demo: break/fix diagnosis.

1. User opens the web UI.
2. User asks: "Why is DNS broken in my k8s-snap cluster?"
3. If no lab exists, `k8s-doc` creates a one-control-plane LXD lab.
4. `k8s-doc` installs k8s from `latest/stable`, bootstraps the node, and verifies baseline state.
5. A DNS failure is introduced by a deterministic playbook or selected from a pre-broken persistent lab.
6. The doctor retrieves relevant k8s-snap and upstream Kubernetes DNS docs.
7. The doctor collects live evidence:
   - `k8s status`
   - `k8s kubectl get nodes`
   - `k8s kubectl get pods -A`
   - CoreDNS pods, service, endpoints, logs, events, and ConfigMap
   - DNS probe from a temporary workload
8. The doctor explains the likely cause using observed evidence and docs citations.
9. The doctor applies a repair inside the disposable lab.
10. The doctor verifies recovery using a DNS probe.
11. The final answer includes summary, evidence, docs references, commands/tools run, fix applied, and verification result.

## Cluster topology

The topology is configurable. Default for MVP:

- one LXD instance
- one k8s-snap control-plane node
- persistent lab by default for demos
- ephemeral mode available for clean runs

Future topology options:

- one control-plane plus one worker
- three control-plane HA cluster
- named scenario-specific labs

## Documentation/RAG design

### Sources

MVP sources:

1. Local k8s-snap docs from this checkout, especially `docs/canonicalk8s/` and selected top-level project docs.
2. Upstream Kubernetes docs from a configurable path or URL. The demo can use a curated local subset to avoid ingesting too much noisy content.

### Retrieval

The MVP uses vector embeddings and local vector search. The design should still keep retrieval behind interfaces so BM25 or hybrid search can be added later.

Core concepts:

- `DocumentSource`: configured source of markdown/html/text docs.
- `Chunker`: converts documents into citation-friendly chunks.
- `Embedder`: creates embedding vectors.
- `VectorStore`: stores and searches chunks.
- `Retriever`: returns ranked chunks for a question.

### Citations

Every retrieved chunk should preserve:

- source name
- file path or URL
- heading path if available
- line range if local markdown supports it
- text snippet

The final answer should include citations in a predictable section.

## LLM provider design

The MVP supports provider abstraction:

- OpenAI-compatible HTTP API
- Ollama-compatible local API

The doctor layer should not depend directly on one provider. It should call an interface such as:

```go
type ChatModel interface {
    Complete(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
```

Provider configuration comes from environment variables and/or config file.

## Tool model

Tools are structured, auditable, and future MCP-compatible.

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() JSONSchema
    Execute(ctx context.Context, input json.RawMessage) (ToolResult, error)
}
```

Each tool invocation records:

- timestamp
- conversation/session ID
- tool name
- validated input
- underlying commands, if any
- stdout/stderr summaries
- exit status
- duration
- result classification

The MVP can mutate freely because the lab is disposable. The audit log is still required for trust, debugging, and demo explainability.

## High-level MVP tools

### Lab tools

- `lab_create`: create or reuse a named LXD lab.
- `lab_status`: show lab nodes and lifecycle status.
- `lab_destroy`: delete lab instances and associated state.
- `lab_reset`: destroy and recreate lab.

### Cluster tools

- `cluster_bootstrap`: install and bootstrap k8s-snap on the control-plane node.
- `cluster_status`: run `k8s status` and parse relevant state.
- `cluster_smoke_test`: run `k8s kubectl get nodes` and `k8s kubectl get pods -A`.

### Docs tools

- `docs_search`: retrieve relevant k8s-snap and upstream Kubernetes chunks.
- `docs_sources`: list configured documentation sources and index status.
- `docs_reindex`: rebuild local vector index.

### DNS diagnostic tools

- `dns_collect`: collect CoreDNS pods, service, endpoints, logs, events, and ConfigMap.
- `dns_break`: intentionally introduce the MVP DNS failure in the disposable lab.
- `dns_repair`: apply the known repair for the MVP DNS failure.
- `dns_verify`: run a DNS probe workload and report success/failure.

## First diagnostic domain: DNS/CoreDNS

DNS is the first deep end-to-end domain because it is visible, Kubernetes-generic, and relevant to k8s-snap.

A practical MVP failure scenario:

1. Baseline cluster DNS works.
2. A playbook breaks CoreDNS by scaling deployment to zero or applying an invalid CoreDNS ConfigMap.
3. The doctor observes that pods cannot resolve `kubernetes.default`.
4. The doctor inspects CoreDNS deployment/pods/service/endpoints/logs/config.
5. The doctor repairs the issue by restoring scale or ConfigMap.
6. The doctor verifies resolution from a temporary pod.

Scaling CoreDNS to zero is the simpler and safer first failure because the diagnosis is easy to explain and the fix is deterministic. Invalid ConfigMap can be a stretch scenario.

## Web UI design

MVP web UI:

- single-page chat interface
- left panel for lab status and actions
- main panel for chat transcript
- expandable sections for evidence, citations, and audit log
- action buttons for common demo operations:
  - create/reuse lab
  - break DNS
  - diagnose DNS
  - repair and verify
  - destroy lab

The backend serves static assets and JSON endpoints. The UI can be plain HTML/CSS/JavaScript to minimize framework overhead.

## Answer format

Responses should be concise first, detailed second:

1. Summary
2. Diagnosis
3. Evidence observed
4. Fix applied or recommended
5. Verification result
6. Docs references
7. Commands/tools run

## Persistence

The MVP supports both modes:

- persistent named lab: default for hackathon demos to avoid repeated setup delays
- ephemeral lab: create, use, and destroy for clean runs

State stored locally under `hackathon/k8s-doc/.state/` or a configurable directory:

- lab metadata
- node inventory
- docs index metadata
- audit logs
- session transcripts

Generated state should be gitignored.

## Configuration

Configuration sources, in priority order:

1. command-line flags
2. environment variables
3. config file
4. defaults

Important settings:

- LXD remote name
- LXD image
- lab name
- persistence mode
- snap channel, default `latest/stable`
- docs source paths/URLs
- embedding provider/model
- chat provider/model
- state directory

## Error handling principles

1. Tool failures become structured results, not panics.
2. Underlying command stderr is captured and summarized.
3. User-visible errors explain what failed and what can be retried.
4. Lab operations are idempotent where practical.
5. Cleanup should be best effort and should not hide the original error.
6. The doctor should distinguish docs-based uncertainty from observed cluster facts.

## Testing strategy

### Unit tests

- config loading
- command construction without executing commands
- audit log writing
- document chunking
- retrieval ranking with fake embeddings
- tool input validation
- doctor answer formatting

### Integration tests with fakes

- fake LXD runner for lab creation flow
- fake k8s/kubectl runner for cluster smoke tests
- fake model provider for deterministic chat responses
- fake vector store for deterministic docs retrieval

### Optional real LXD tests

Real LXD tests should be opt-in through environment variables because they are slow and mutate local system state.

## Security and safety

MVP is intentionally allowed to mutate the disposable lab. Still:

- The LLM sees high-level tools, not arbitrary shell.
- Low-level command execution is allowlisted.
- LXD instance names are generated under a project prefix.
- File push/pull paths are validated.
- Audit logs capture all tool calls.
- Future remote-cluster support must introduce read-only and approval-gated policies.

## Future work

1. Add MCP adapter around the internal tool registry.
2. Support remote user clusters with strict read-only default policy.
3. Add more diagnostic domains: LoadBalancer/MetalLB, Gateway/Ingress, node readiness, storage, pod scheduling.
4. Add hybrid retrieval with BM25 plus vectors.
5. Add source management UI for custom docs.
6. Add scenario packs for QA teams.
7. Add report export: markdown diagnosis bundle with audit log and citations.
8. Add multi-node and HA lab presets.

## MVP acceptance criteria

The MVP is complete when:

1. `hackathon/k8s-doc` builds as a standalone Go project.
2. Web UI starts locally.
3. Docs from k8s-snap and a configured upstream Kubernetes source can be indexed.
4. A user can ask a question and receive a cited answer.
5. The app can create or reuse a persistent local LXD lab.
6. The app installs `k8s` from `latest/stable` and bootstraps a one-node cluster.
7. The app can run `k8s status` and `k8s kubectl get nodes` through typed tools.
8. The DNS failure scenario can be introduced.
9. The doctor can collect DNS diagnostics, identify the issue, apply the fix, and verify recovery.
10. Every tool call and underlying command is recorded in an audit log.
