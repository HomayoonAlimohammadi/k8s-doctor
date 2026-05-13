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
