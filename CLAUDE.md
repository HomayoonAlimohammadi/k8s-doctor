# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`k8s-doc` (binary name) / `k8s-doctor` (module) is a Go hackathon prototype: an AI-assisted Kubernetes troubleshooter that drives a disposable LXD-backed `k8s` snap lab, runs constrained diagnostic playbooks against it, and returns RAG-grounded answers. The canonical design lives in `docs/design.md`; the README is a short pitch.

## Commands

Run the web server (default subcommand; serves UI + chat API on `K8S_DOC_HTTP_ADDR`, default `127.0.0.1:8080`):

```bash
go run ./cmd/k8s-doc serve
```

Re-index documentation into the in-memory RAG store (reads `K8S_DOC_K8S_SNAP_DOCS` and `K8S_DOC_UPSTREAM_K8S_DOCS`):

```bash
go run ./cmd/k8s-doc reindex
```

Tests:

```bash
go test ./...                              # full unit suite (LXD integration test skipped by default)
go test ./internal/playbooks -run TestDNS  # single package / single test
K8S_DOC_RUN_LXD_TESTS=1 go test ./internal/lab -run TestRealLXDCreateDestroy  # real LXD; needs lxd installed
```

`go vet ./...` and `gofmt -l .` are the lint baseline. There is no Makefile.

Important runtime detail: `internal/web/server.go` serves `web/static/index.html` and `/static/` from the **current working directory**. Run the binary from the repo root, or static assets 404.

## Architecture

Single binary, dispatch in `cmd/k8s-doc/main.go`. `run()` wires every package together — read it first when tracing what gets instantiated. The composition root is intentionally explicit (no DI framework).

Layering (LLM sees only the top layer; lower layers are implementation details):

1. **`internal/web`** — `net/http.ServeMux`. Routes: `/api/chat`, `/api/lab/{create,destroy,status}`, `/api/dns/break`, `/api/health`. `LabService` and `DoctorService` are interfaces; `RealLabService` / `RealDoctorService` in `services.go` wrap the real implementations and write audit entries.
2. **`internal/doctor`** — orchestration. `Doctor.DiagnoseDNS` calls `Retriever.Search` (docs) + `DNSDiagnostic.Collect` (live evidence), then `FormatAnswer` produces the canonical sectioned response (Summary / Diagnosis / Evidence / Fix / Verification / Citations / Tools run). `KubectlDNSDiagnostic` adapts a `playbooks.DNSPlaybook` to the `DNSDiagnostic` interface.
3. **`internal/playbooks`** — deterministic diagnostic/repair flows. `DNSPlaybook` depends only on a narrow `Kubectl` interface (Get/Describe/Logs/ApplyYAML/Scale/RunDNSProbe). Add new playbooks here, not in `doctor`.
4. **`internal/tools/kubectl`, `internal/tools/k8ssnap`, `internal/tools/lxd`** — typed adapters that shell out via `tools.Runner`. `kubectl` operates **inside** an LXD instance (`lxc exec <node> -- k8s kubectl ...`); it does not run on the host. The `lxd` client wraps `lxc launch/delete/exec`.
5. **`internal/tools/runner.go`** — `ExecRunner` (real `os/exec`) and `FakeRunner` (captures args, returns canned result). Every test below the `tools` boundary uses `FakeRunner` — do the same when adding tests; do not shell out from unit tests.
6. **`internal/tools/registry.go`** — `Tool` interface + `Registry` for LLM-exposed tools with JSON schemas. Currently scaffolded but the `serve` path does not yet drive tool calls through it; the MVP wires the doctor directly.
7. **`internal/lab`** — `Manager` owns lab lifecycle. State (node names + roles) is persisted as JSON to `${StateDir}/lab.json`. `Backend` is the only interface; `cmd/k8s-doc/main.go` adapts the `lxd.Client` to it via an anonymous wrapper.
8. **`internal/rag`** — `MemoryIndex` is an in-memory cosine-similarity store. `Reindexer` walks `Source` implementations (currently directory sources) and chunks via `chunk.go`. The `Embedder` interface is satisfied by `llm.FakeEmbeddingModel` in the default wiring — real OpenAI/Ollama embedders exist in `internal/llm` but are not yet wired into `main.go`.
9. **`internal/llm`** — `ChatModel` and `EmbeddingModel` interfaces with `openai.go`, `ollama.go`, and `fake.go` implementations. Provider selection is config-driven (`K8S_DOC_CHAT_PROVIDER`, `K8S_DOC_EMBEDDING_PROVIDER`) but `main.go` currently hard-codes `FakeEmbeddingModel`; adding real-model wiring is an open task.
10. **`internal/audit`** — append-only JSONL at `${StateDir}/audit.jsonl`. `services.go` wrappers in `internal/web` are responsible for recording entries; if you add a new tool-using path, wrap it in audit too.
11. **`internal/config`** — every setting is an env var with a default; see `config.go`. There is no config file format.

### Conventions specific to this repo

- All cross-package collaborators are **interfaces defined in the consumer package** (e.g. `doctor.Retriever`, `playbooks.Kubectl`, `lab.Backend`). When adding a dependency, define the interface where it is used, then satisfy it from the provider package. This is what keeps the LLM/tool boundary clean.
- Each interface has a `Fake*` (or `FakeRunner`-style) sibling next to the production implementation; tests consume the fakes.
- Errors are wrapped with `fmt.Errorf("...: %w", err)` at every package boundary; preserve this when adding code.
- The `serve` binary expects to run from the repo root because of the `web/static` file server — do not change that without also fixing the static handler.
- `${StateDir}` (default `.state/`) is the single sink for runtime state (`lab.json`, `audit.jsonl`, downloaded upstream docs). The `.state/` directory is in `.gitignore`.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **k8s-doctor** (775 symbols, 1683 relationships, 24 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/k8s-doctor/context` | Codebase overview, check index freshness |
| `gitnexus://repo/k8s-doctor/clusters` | All functional areas |
| `gitnexus://repo/k8s-doctor/processes` | All execution flows |
| `gitnexus://repo/k8s-doctor/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.opencode/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.opencode/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.opencode/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.opencode/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.opencode/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.opencode/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
